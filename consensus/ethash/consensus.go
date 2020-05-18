// Copyright 2018  The go-genchain Authors
// This file is part of the go-genchain library.
//
// The go-genchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-genchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-genchain library. If not, see <http://www.gnu.org/licenses/>.

package ethash

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/genchain/go-genchain/common"
	"github.com/genchain/go-genchain/common/math"
	"github.com/genchain/go-genchain/consensus"
	"github.com/genchain/go-genchain/consensus/misc"
	"github.com/genchain/go-genchain/core/state"
	"github.com/genchain/go-genchain/core/types"
	"github.com/genchain/go-genchain/params"
)

// proof-of-work protocol constants.
var (
	GenBlockReward         *big.Int = big.NewInt(4.5e+17)
	GenBlockUncleReward    *big.Int = big.NewInt(5e+16)
	GenBlockEcoReward      *big.Int = big.NewInt(1e+15)
	TotalCoin              *big.Int = big.NewInt(1.33e+7)
	maxUncles                       = 5                // Maximum number of uncles allowed in a single block
	allowedFutureBlockTime          = 12 * time.Second // Max time from current time allowed for blocks, before they're considered future blocks
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	errLargeBlockTime    = errors.New("timestamp too big")
	errZeroBlockTime     = errors.New("timestamp equals parent's")
	errTooManyUncles     = errors.New("too many uncles")
	errDuplicateUncle    = errors.New("duplicate uncle")
	errUncleIsAncestor   = errors.New("uncle is ancestor")
	errDanglingUncle     = errors.New("uncle's parent is not ancestor")
	errInvalidDifficulty = errors.New("non-positive difficulty")
	errInvalidMixDigest  = errors.New("invalid mix digest")
	errInvalidPoW        = errors.New("invalid proof-of-work")
)

// Author implements consensus.Engine, returning the header's coinbase as the
// proof-of-work verified author of the block.
func (ethash *Ethash) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

// VerifyHeader checks whether a header conforms to the consensus rules of the
// stock Ethereum ethash engine.
func (ethash *Ethash) VerifyHeader(chain consensus.ChainReader, header *types.Header, seal bool) error {
	// If we're running a full engine faking, accept any input as valid
	if ethash.config.PowMode == ModeFullFake {
		return nil
	}
	// Short circuit if the header is known, or it's parent not
	number := header.Number.Uint64()
	if chain.GetHeader(header.Hash(), number) != nil {
		return nil
	}
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	// Sanity checks passed, do a proper verification
	return ethash.verifyHeader(chain, header, parent, false, seal)
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers
// concurrently. The method returns a quit channel to abort the operations and
// a results channel to retrieve the async verifications.
func (ethash *Ethash) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	// If we're running a full engine faking, accept any input as valid
	if ethash.config.PowMode == ModeFullFake || len(headers) == 0 {
		abort, results := make(chan struct{}), make(chan error, len(headers))
		for i := 0; i < len(headers); i++ {
			results <- nil
		}
		return abort, results
	}

	// Spawn as many workers as allowed threads
	workers := runtime.GOMAXPROCS(0)
	if len(headers) < workers {
		workers = len(headers)
	}

	// Create a task channel and spawn the verifiers
	var (
		inputs = make(chan int)
		done   = make(chan int, workers)
		errors = make([]error, len(headers))
		abort  = make(chan struct{})
	)
	for i := 0; i < workers; i++ {
		go func() {
			for index := range inputs {
				errors[index] = ethash.verifyHeaderWorker(chain, headers, seals, index)
				done <- index
			}
		}()
	}

	errorsOut := make(chan error, len(headers))
	go func() {
		defer close(inputs)
		var (
			in, out = 0, 0
			checked = make([]bool, len(headers))
			inputs  = inputs
		)
		for {
			select {
			case inputs <- in:
				if in++; in == len(headers) {
					// Reached end of headers. Stop sending to workers.
					inputs = nil
				}
			case index := <-done:
				for checked[index] = true; checked[out]; out++ {
					errorsOut <- errors[out]
					if out == len(headers)-1 {
						return
					}
				}
			case <-abort:
				return
			}
		}
	}()
	return abort, errorsOut
}

func (ethash *Ethash) verifyHeaderWorker(chain consensus.ChainReader, headers []*types.Header, seals []bool, index int) error {
	var parent *types.Header
	if index == 0 {
		parent = chain.GetHeader(headers[0].ParentHash, headers[0].Number.Uint64()-1)
	} else if headers[index-1].Hash() == headers[index].ParentHash {
		parent = headers[index-1]
	}
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	if chain.GetHeader(headers[index].Hash(), headers[index].Number.Uint64()) != nil {
		return nil // known block
	}
	return ethash.verifyHeader(chain, headers[index], parent, false, seals[index])
}

// VerifyUncles verifies that the given block's uncles conform to the consensus
// rules of the stock Ethereum ethash engine.
func (ethash *Ethash) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	// If we're running a full engine faking, accept any input as valid
	if ethash.config.PowMode == ModeFullFake {
		return nil
	}
	// Verify that there are at most 2 uncles included in this block
	if len(block.Uncles()) > maxUncles {
		return errTooManyUncles
	}
	// Gather the set of past uncles and ancestors
	uncles, ancestors := mapset.NewSet(), make(map[common.Hash]*types.Header) //set.New(), make(map[common.Hash]*types.Header)

	number, parent := block.NumberU64()-1, block.ParentHash()
	for i := 0; i < 7; i++ {
		ancestor := chain.GetBlock(parent, number)
		if ancestor == nil {
			break
		}
		ancestors[ancestor.Hash()] = ancestor.Header()
		for _, uncle := range ancestor.Uncles() {
			uncles.Add(uncle.Hash())
		}
		parent, number = ancestor.ParentHash(), number-1
	}
	ancestors[block.Hash()] = block.Header()
	uncles.Add(block.Hash())

	// Verify each of the uncles that it's recent, but not an ancestor
	for _, uncle := range block.Uncles() {
		// Make sure every uncle is rewarded only once
		hash := uncle.Hash()
		if uncles.Contains(hash) {
			return errDuplicateUncle
		}
		uncles.Add(hash)

		// Make sure the uncle has a valid ancestry
		if ancestors[hash] != nil {
			return errUncleIsAncestor
		}
		if ancestors[uncle.ParentHash] == nil || uncle.ParentHash == block.ParentHash() {
			return errDanglingUncle
		}
		if err := ethash.verifyHeader(chain, uncle, ancestors[uncle.ParentHash], true, true); err != nil {
			return err
		}
	}
	return nil
}

// verifyHeader checks whether a header conforms to the consensus rules of the
// stock Ethereum ethash engine.
// See YP section 4.3.4. "Block Header Validity"
func (ethash *Ethash) verifyHeader(chain consensus.ChainReader, header, parent *types.Header, uncle bool, seal bool) error {
	// Ensure that the header's extra-data section is of a reasonable size
	if uint64(len(header.Extra)) > params.MaximumExtraDataSize {
		return fmt.Errorf("extra-data too long: %d > %d", len(header.Extra), params.MaximumExtraDataSize)
	}
	// Verify the header's timestamp
	if uncle {
		if header.Time.Cmp(math.MaxBig256) > 0 {
			return errLargeBlockTime
		}
	} else {
		if header.Time.Cmp(big.NewInt(time.Now().Add(allowedFutureBlockTime).Unix())) > 0 {
			return consensus.ErrFutureBlock
		}
	}
	if header.Time.Cmp(parent.Time) <= 0 {
		return errZeroBlockTime
	}
	// Verify the block's difficulty based in it's timestamp and parent's difficulty
	expected := ethash.CalcDifficulty(chain, header.Time.Uint64(), parent)

	if expected.Cmp(header.Difficulty) != 0 {
		return fmt.Errorf("invalid difficulty: have %v, want %v", header.Difficulty, expected)
	}

	// Verify that the gas limit is <= 2^63-1
	cap := uint64(0x7fffffffffffffff)
	if header.GasLimit > cap {
		return fmt.Errorf("invalid gasLimit: have %v, max %v", header.GasLimit, cap)
	}
	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("invalid gasUsed: have %d, gasLimit %d", header.GasUsed, header.GasLimit)
	}

	// Verify that the gas limit remains within allowed bounds
	diff := int64(parent.GasLimit) - int64(header.GasLimit)
	if diff < 0 {
		diff *= -1
	}
	limit := parent.GasLimit / params.GasLimitBoundDivisor

	if uint64(diff) >= limit || header.GasLimit < params.MinGasLimit {
		return fmt.Errorf("invalid gas limit: have %d, want %d += %d", header.GasLimit, parent.GasLimit, limit)
	}
	// Verify that the block number is parent's +1
	if diff := new(big.Int).Sub(header.Number, parent.Number); diff.Cmp(big.NewInt(1)) != 0 {
		return consensus.ErrInvalidNumber
	}
	// Verify the engine specific seal securing the block
	if seal {
		if err := ethash.VerifySeal(chain, header); err != nil {
			return err
		}
	}
	// If all checks passed, validate any special fields for hard forks
	if err := misc.VerifyDAOHeaderExtraData(chain.Config(), header); err != nil {
		return err
	}
	if err := misc.VerifyForkHashes(chain.Config(), header, uncle); err != nil {
		return err
	}
	return nil
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns
// the difficulty that a new block should have when created at time
// given the parent block's time and difficulty.
func (ethash *Ethash) CalcDifficulty(chain consensus.ChainReader, time uint64, parent *types.Header) *big.Int {

	return CalcDifficulty(chain.Config(), time, parent)
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns
// the difficulty that a new block should have when created at time
// given the parent block's time and difficulty.
func CalcDifficulty(config *params.ChainConfig, time uint64, parent *types.Header) *big.Int {
	next := new(big.Int).Add(parent.Number, big1)
	switch {
	case config.IsByzantium(next):
		return calcDifficultyByzantium(time, parent)
	case config.IsHomestead(next):
		return calcDifficultyHomestead(time, parent)
	default:
		return calcDifficultyFrontier(time, parent)
	}
}

func calcnp(timespan uint64, n uint64, p uint64) (uint64, uint64) {
	if p < 256 {
		if timespan < 102 {
			p = p + 1
		} else if timespan < 108 {
			n = n + 1
		} else if timespan > 138 {
			if p > params.P {
				p = p - 1
			}
		} else if timespan > 132 {
			if n > params.N {
				n = n - 1
			}
		}
		if p <= params.P && n > params.N && timespan > 360 {
			n = n - 1
		}
	} else {

		if timespan < 102 {
			n = n + 1
		} else if timespan > 138 {
			if n > params.N {
				n = n - 1
			}
		}

		if n <= params.N && p > params.P && timespan > 360 {
			p = p - 1
		}
	}

	if n <= params.N {
		n = params.N
	}

	if p <= params.P {
		p = params.P
	}

	if p > 256 {
		p = 256
	}
	return n, p
}
func (ethash *Ethash) CalcDifficultyByLake(header *types.Header, parent *types.Header, parent12 *types.Header) (uint64, uint64, *big.Int, *big.Int) {
	var n, p uint64
	var timespan uint64 = 120
	Alpha := new(big.Int)
	Alpha.SetUint64(timespan)
	NP := new(big.Int)

	curBlockNumber := header.Number.Uint64() //得到当前区块号
	curBlockTime := header.Time.Uint64()     //得到当前区块的时间

	curNP := new(big.Int) //当前块总难度

	if curBlockNumber < 1 {
		NP.SetUint64(0)
	} else {
		NP.Set(parent.NP) //Get the total difficulty of the parent block
	}

	if curBlockNumber <= 12 {
		curNP.SetUint64(params.N * params.N * params.N * params.P * params.P * params.P * params.P * params.P * params.P) //当前区块的总难度np
		NP.Add(NP, curNP)                                                                                                 //加上当前区块的np
		return params.N, params.P, Alpha, NP
	}

	parent12BlockTime := parent12.Time.Uint64() //Get the time of the first 12 blocks
	timespan = curBlockTime - parent12BlockTime //Time difference from the previous 12 blocks

	if curBlockTime < parent12BlockTime {
		timespan = 120
	}
	Alpha.SetUint64(timespan)

	//If the P value can be adjusted
	n, p = calcnp(timespan, parent.N, parent.P)
	curNP.SetUint64(n * n * n * p * p * p * p * p * p) //The difficulty of the current block
	NP.Add(NP, curNP)                                  //Plus the np of the current block,

	return n, p, Alpha, NP
}
func (ethash *Ethash) CalcDifficultyBygen(header *types.Header, parent *types.Header, parent12 *types.Header) (uint64, uint64, *big.Int, *big.Int) {

	var n, p uint64
	np := new(big.Int)    //totaldifficulty
	alpha := new(big.Int) //timespan
	n, p, alpha, np = ethash.CalcDifficultyByLake(header, parent, parent12)

	return n, p, alpha, np
}

func (ethash *Ethash) VerifyDifficultyBygen(header *types.Header) (uint64, uint64) {

	var timespan uint64
	timespan = header.Alpha.Uint64()
	n, p := calcnp(timespan, header.NN, header.PP)
	return n, p

}

//Change the difficulty adjustment algorithm to sea
func (ethash *Ethash) CalcDifficultyBySea(header *types.Header, parent *types.Header) (uint64, uint64, *big.Int, *big.Int) {

	curBlockNumber := header.Number.Uint64()
	curBlockTime := header.Time.Uint64()

	//Calculate the duration of two blocks
	var timespan uint64 = 10 //default timespan between block
	parentBlockTime := parent.Time.Uint64()
	if curBlockNumber == 1 { //The length of the first block is 10
		timespan = 10
	} else {
		timespan = curBlockTime - parentBlockTime
	}

	if curBlockTime < parentBlockTime {
		timespan = 10
	}
	Alpha := new(big.Int) //timespan
	Alpha.SetUint64(timespan)

	//Calculate n, p
	var n, p uint64
	n, p = calcnpsea(timespan, parent.N, parent.P)

	//Calculate the difficulty of the current block
	curNP := new(big.Int) //Total difficulty of the current block
	curNP.SetUint64(n*n*n*p*p*p*p*p*p - timespan)
	//Calculate the total difficulty
	NP := new(big.Int) //Total difficulty
	if curBlockNumber < 1 {
		NP.SetUint64(0)
	} else {
		NP.Set(parent.NP)
	}

	NP.Add(NP, curNP)

	return n, p, Alpha, NP
}
func calcnpsea(timespan uint64, n uint64, p uint64) (uint64, uint64) {
	if p < 256 {
		if timespan < 5 {
			p = p + 1
		} else if timespan < 7 {
			n = n + 1
		} else if timespan > 900 {
			if p > params.P && n > params.N {
				p = p - p/7
				n = n - n/7
			}
		} else if timespan > 600 {
			if p > params.P && n > params.N {
				p = p - p/10
				n = n - n/10
			}
		} else if timespan > 16 {
			if p > params.P {
				p = p - 1
			}
		} else if timespan > 13 {
			if n > params.N {
				n = n - 1
			}
		}

		if p <= params.P && n > params.N && timespan > 30 {
			n = n - 1
		}
	} else {
		if timespan < 5 {
			n = n + 1
		} else if timespan > 16 {
			if n > params.N {
				n = n - 1
			}
		}

		if n <= params.N && p > params.P && timespan > 30 {
			p = p - 1
		}
	}

	if n <= params.N {
		n = params.N
	}

	if p <= params.P {
		p = params.P
	}

	if p > 256 {
		p = 256
	}
	return n, p
}
func (ethash *Ethash) VerifyDifficultyBySea(header *types.Header) (uint64, uint64) {

	var timespan uint64
	timespan = header.Alpha.Uint64()
	var n, p uint64
	n, p = calcnpsea(timespan, header.NN, header.PP)
	//swpu
	//fmt.Println("verfiy Diffculty  BlockNumer Fork:", header.Number, "calcnpsea  ")

	return n, p

}

// Some weird constants to avoid constant memory allocs for them.
var (
	expDiffPeriod = big.NewInt(100000)
	big1          = big.NewInt(1)
	big2          = big.NewInt(2)
	big9          = big.NewInt(9)
	big10         = big.NewInt(10)
	bigMinus99    = big.NewInt(-99)
	big2999999    = big.NewInt(2999999)
	big_max       = big.NewInt(1e+18)
)

// calcDifficultyByzantium is the difficulty adjustment algorithm. It returns
// the difficulty that a new block should have when created at time given the
// parent block's time and difficulty. The calculation uses the Byzantium rules.
func calcDifficultyByzantium(time uint64, parent *types.Header) *big.Int {
	// https://github.com/ethereum/EIPs/issues/100.
	// algorithm:
	// diff = (parent_diff +
	//         (parent_diff / 2048 * max((2 if len(parent.uncles) else 1) - ((timestamp - parent.timestamp) // 9), -99))
	//        ) + 2^(periodCount - 2)

	bigTime := new(big.Int).SetUint64(time)
	bigParentTime := new(big.Int).Set(parent.Time)

	// holds intermediate values to make the algo easier to read & audit
	x := new(big.Int)
	y := new(big.Int)

	// (2 if len(parent_uncles) else 1) - (block_timestamp - parent_timestamp) // 9
	x.Sub(bigTime, bigParentTime)
	x.Div(x, big9)
	if parent.UncleHash == types.EmptyUncleHash {
		x.Sub(big1, x)
	} else {
		x.Sub(big2, x)
	}
	// max((2 if len(parent_uncles) else 1) - (block_timestamp - parent_timestamp) // 9, -99)
	if x.Cmp(bigMinus99) < 0 {
		x.Set(bigMinus99)
	}
	// parent_diff + (parent_diff / 2048 * max((2 if len(parent.uncles) else 1) - ((timestamp - parent.timestamp) // 9), -99))
	y.Div(parent.Difficulty, params.DifficultyBoundDivisor)
	x.Mul(y, x)
	x.Add(parent.Difficulty, x)

	// minimum difficulty can ever be (before exponential factor)
	if x.Cmp(params.MinimumDifficulty) < 0 {
		x.Set(params.MinimumDifficulty)
	}
	// calculate a fake block number for the ice-age delay:
	//   https://github.com/ethereum/EIPs/pull/669
	//   fake_block_number = min(0, block.number - 3_000_000
	fakeBlockNumber := new(big.Int)
	if parent.Number.Cmp(big2999999) >= 0 {
		fakeBlockNumber = fakeBlockNumber.Sub(parent.Number, big2999999) // Note, parent is 1 less than the actual block number
	}
	// for the exponential factor
	periodCount := fakeBlockNumber
	periodCount.Div(periodCount, expDiffPeriod)

	// the exponential factor, commonly referred to as "the bomb"
	// diff = diff + 2^(periodCount - 2)
	if periodCount.Cmp(big1) > 0 {
		y.Sub(periodCount, big2)
		y.Exp(big2, y, nil)
		x.Add(x, y)
	}
	return x
}

// calcDifficultyHomestead is the difficulty adjustment algorithm. It returns
// the difficulty that a new block should have when created at time given the
// parent block's time and difficulty. The calculation uses the Homestead rules.
func calcDifficultyHomestead(time uint64, parent *types.Header) *big.Int {
	// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-2.md
	// algorithm:
	// diff = (parent_diff +
	//         (parent_diff / 2048 * max(1 - (block_timestamp - parent_timestamp) // 10, -99))
	//        ) + 2^(periodCount - 2)

	bigTime := new(big.Int).SetUint64(time)
	bigParentTime := new(big.Int).Set(parent.Time)

	// holds intermediate values to make the algo easier to read & audit
	x := new(big.Int)
	y := new(big.Int)

	// 1 - (block_timestamp - parent_timestamp) // 10
	x.Sub(bigTime, bigParentTime)
	x.Div(x, big10)
	x.Sub(big1, x)

	// max(1 - (block_timestamp - parent_timestamp) // 10, -99)
	if x.Cmp(bigMinus99) < 0 {
		x.Set(bigMinus99)
	}
	// (parent_diff + parent_diff // 2048 * max(1 - (block_timestamp - parent_timestamp) // 10, -99))
	y.Div(parent.Difficulty, params.DifficultyBoundDivisor)
	x.Mul(y, x)
	x.Add(parent.Difficulty, x)

	// minimum difficulty can ever be (before exponential factor)
	if x.Cmp(params.MinimumDifficulty) < 0 {
		x.Set(params.MinimumDifficulty)
	}
	// for the exponential factor
	periodCount := new(big.Int).Add(parent.Number, big1)
	periodCount.Div(periodCount, expDiffPeriod)

	// the exponential factor, commonly referred to as "the bomb"
	// diff = diff + 2^(periodCount - 2)
	if periodCount.Cmp(big1) > 0 {
		y.Sub(periodCount, big2)
		y.Exp(big2, y, nil)
		x.Add(x, y)
	}
	return x
}

// calcDifficultyFrontier is the difficulty adjustment algorithm. It returns the
// difficulty that a new block should have when created at time given the parent
// block's time and difficulty. The calculation uses the Frontier rules.
func calcDifficultyFrontier(time uint64, parent *types.Header) *big.Int {
	diff := new(big.Int)
	adjust := new(big.Int).Div(parent.Difficulty, params.DifficultyBoundDivisor)
	bigTime := new(big.Int)
	bigParentTime := new(big.Int)

	bigTime.SetUint64(time)
	bigParentTime.Set(parent.Time)

	if bigTime.Sub(bigTime, bigParentTime).Cmp(params.DurationLimit) < 0 {
		diff.Add(parent.Difficulty, adjust)
	} else {
		diff.Sub(parent.Difficulty, adjust)
	}
	if diff.Cmp(params.MinimumDifficulty) < 0 {
		diff.Set(params.MinimumDifficulty)
	}

	periodCount := new(big.Int).Add(parent.Number, big1)
	periodCount.Div(periodCount, expDiffPeriod)
	if periodCount.Cmp(big1) > 0 {
		// diff = diff + 2^(periodCount - 2)
		expDiff := periodCount.Sub(periodCount, big2)
		expDiff.Exp(big2, expDiff, nil)
		diff.Add(diff, expDiff)
		diff = math.BigMax(diff, params.MinimumDifficulty)
	}
	return diff
}

// VerifySeal implements consensus.Engine, checking whether the given block satisfies
// the PoW difficulty requirements.
func (ethash *Ethash) VerifySeal(chain consensus.ChainReader, header *types.Header) error {
	// If we're running a fake PoW, accept any seal as valid
	if ethash.config.PowMode == ModeFake || ethash.config.PowMode == ModeFullFake {
		time.Sleep(ethash.fakeDelay)
		if ethash.fakeFail == header.Number.Uint64() {
			return errInvalidPoW
		}
		return nil
	}
	// If we're running a shared PoW, delegate verification to it
	if ethash.shared != nil {
		//fmt.Println("GPU consensus.go VerifySeal() share pow ")
		return ethash.shared.VerifySeal(chain, header)
	}
	// Ensure that we have a valid difficulty for the block
	if header.Difficulty.Sign() <= 0 {
		return errInvalidDifficulty
	}
	//verify fhash,hash256 is Ok
	hash := header.HashNoNonce().Bytes()
	nonce := header.Nonce.Uint64()

	fhash, _, hash256 := genHash(hash, nonce, hash, header.P, header.N)
	fhashstring := common.BytesToHash(fhash).String()

	//verify fash
	if header.FuzzyHash.String() != fhashstring {
		return errInvalidMixDigest
	}
	//verify hash nonce

	P := int(header.P)
	if !compareDiff(hash256, P) {
		return errInvalidMixDigest
	}
	//verify n,p is ok
	var n, p uint64

	next := new(big.Int).Set(header.Number)
	if chain.Config().IsSeafork(next) {
		n, p = ethash.VerifyDifficultyBySea(header)
	} else {
		n, p = ethash.VerifyDifficultyBygen(header)
	}

	if header.N != n || header.P != p {

		return errInvalidPoW
	}
	return nil
}

// Prepare implements consensus.Engine, initializing the difficulty field of a
// header to conform to the ethash protocol. The changes are done inline.
func (ethash *Ethash) Prepare(chain consensus.ChainReader, header *types.Header) error {
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	//gen begin   Counting from four blocks to 12 blocks
	if parent.Time.Cmp(header.Time) >= 0 { //No calculation
		return consensus.ErrUnknownAncestor
	}

	var (
		n     uint64
		p     uint64
		alpha *big.Int
		np    *big.Int
	)

	next := new(big.Int).Add(parent.Number, big1)
	if chain.Config().IsSeafork(next) {
		n, p, alpha, np = ethash.CalcDifficultyBySea(header, parent)
	} else {
		var parent12 *types.Header
		if header.Number.Uint64() >= 13 {
			parent12 = chain.GetHeaderByNumber(header.Number.Uint64() - 12)
		}
		n, p, alpha, np = ethash.CalcDifficultyBygen(header, parent, parent12)
	}

	header.N = n
	header.NN = parent.N
	header.P = p
	header.PP = parent.P
	header.Alpha.Set(alpha) //timespan
	header.NP.Set(np)       //totalDiffcult
	//gen end

	header.Difficulty = ethash.CalcDifficulty(chain, header.Time.Uint64(), parent)

	return nil
}

// Finalize implements consensus.Engine, accumulating the block and uncle rewards,
// setting the final state and assembling the block.
func (ethash *Ethash) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	// Accumulate any block and uncle rewards and commit the final state root
	accumulateRewardsGen(chain.Config(), state, header, uncles)
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))

	// Header seems complete, assemble into a block and return
	return types.NewBlock(header, txs, uncles, receipts), nil
}

// Some weird constants to avoid constant memory allocs for them.
var (
	big5  = big.NewInt(5)
	big6  = big.NewInt(6)
	big8  = big.NewInt(8)
	big32 = big.NewInt(32)

	CDAddress = []string{
		"0x49ff31917cd16c593d376347f82f7ea67a7ded0d",
		"0x6e2aeaa5d6bbd27656aa8c774005e71d9afc1b23",
		"0x80960290c3e717ba425333219e2b4a64c9184422",
		"0xde0e25c523a107fc71a955288e95fc80e74d114b",
		"0x6c8df9d21c7087125f448016a2f2afcc14bb8c32",
		"0xc21581f15ffe2da6ac5e2efc04cefd5f6ba8c121",
		"0xaf524d5a4aedef7e4ab6580b68f4bbfcd7ed9064",
		"0x8f1eeeade57c518f561169e9e473b6737410106d",
		"0x283d14e63bb224923d92c0a3e20d8d0f8554fdc6",
		"0x1edc6edcb4456badbe2f84ea2868439467303f39",
		"0x35a93c4ba8ae10156950a9a760a922b990223f7e",
		"0xf7922e6085dbb8f9af7b998647bf52a8a67323ea",
		"0xaea94a6c6436c181e976423fca23a2fc58ff0e0e",
		"0xbff99ec8cbf9cd3d27a5a41ab22bf9a1841b658c",
		"0x5e4d01a8b2f4f4385a396f0276090ce9ba70fbec",
		"0xf72afb5b6b87516b96440665b0efef3b466f2c8f",
		"0xc970baa3fe0f050628803560c4f4763a8cb89641",
		"0x72508937ac5d4ea2dfdce7885480eca36a4a23fa",
		"0x9f8136bd79512e809f90ce0c1e451b0d6991aa63",
		"0xa4f36136865312bb5e0d42aed529126f09bb1b02",
		"0x3df38e8fbf2bc869afdec75ceeb25cf470b047e1",
		"0xbc88ffcc81bdb180f74a0590e8586d147ed1ed85",
		"0x771554d5a2cb453f4ed459b830ed4011fb8ce68a",
		"0x127426bed3724449b9efc1da7058f00498c0338d",
		"0x4e1a6355a35466b6cc1b02492795814128a56799",
		"0xd934cdf46a7ac61ce91ebaa92bc20afb68c9b566",
		"0xac2bcd2ed9876051d4c64dd899d38f95e68bdce7",
		"0x6f20fddfdbb96b9516dd9b75fc54c75595581cb3",
		"0x39136041c26225e97dc55bf897881280258722ea",
		"0x9ab1c0ed107c5e3521ca017f3011cbd6cf856202",
		"0x7520afef96fecc57884449b14beca134cbafbed8",
		"0x78b0472be31df30b4c02f83108661f1adc99abd7",
		"0xfd5da58f901548cb0e06a0d74c3d3f9dead8831f",
		"0x17c38c7c4258d9bb75165c828ed0394933b87b28",
		"0x0a50575359efbad65c4c68f71906d663185138bc",
		"0x59f8c5d60d80dcd06add171a09182f9764e58e6a",
		"0x01a24c4e8b82b3c1838d9f4d8b6a8070eeae06b3",
		"0x589ab7907f14d05488c029a362b5f1aeeb9d2d3e",
		"0x201a14780fef99e5793b2da30b4cc5d41c3d51e8",
		"0xde4597c58fa29b7642c317c9a6575fcea8c8f32a",
		"0x73cd1b163c038629cf57987405dfaf964452024a",
		"0x7ebc3ea0ec38c99d76d13cc618760a28a241910f",
		"0xff344df8352209e4a841a95d890695de45dfdfb2",
		"0xbaaa990b7abeb0fc3587dfb446c9f27897daaa07",
		"0x1d07362846ab350377de07efa65126671354716b",
		"0x461a2f4ae6d1651a5279d4d551dc457d6ddce9e1",
		"0xda1a349c67e15c0cdeaeef1cf54041a37652f86e",
		"0x70a7bafcd8b9bfdd1bad2a1099acdc324eebd816",
		"0x4e9ff86bcbfab42d07cc143a8d1775c94010a9bb",
		"0x1cebf431b95076254687a385ccb03aca80c3d543",
		"0x0ebd62ba3e7dea2fef3c583cab94ba32271cfad9",
		"0x86a08724ca02071a93401428bd5c37e827db8c1b",
		"0x0932388681886fe81dce06fa5b50ecda0af6d22b",
		"0x011ed29043ecfe7176ab06879f9475dab260e3f8",
		"0x4e6d3140de836c33828d7cafabbb24b0a0263bc7",
		"0x7cc171a2018dc46a3b22e8905811e31a508cdc5a",
		"0x0911306e8e46bf03e862c0ca39e9eb4d9f175527",
		"0x9bb2fcfe40cfe79788d9daf654f9e5e660376880",
		"0xfa3959dd6925bdf634a19fcf9fc9a74a852b02f1",
		"0x72165c2c6ef16d8b972567d4e4c45f8cfd2f13c3",
		"0x45e1e09ba41644465532b5ed6c439ac6dfe23f59",
		"0x841111d1fe42be7b96e6689e9c94497dc32e3c9d",
		"0x353ae2b4bc037d15e4d08ec2fa18907514e019a2",
		"0x19c4b8d1d4a4d20f0b56163c169f60a851f2956c",
		"0xc02cb1a2ca0f72a5fc0f9798035bf62c381d8e11",
		"0x020758e61bbb5fa332f2c67f0e031d6fcadd6149",
		"0xfffee9d11fb0dd82a57013c74299d604b0bb753e",
		"0x5d1d57a929edf499f0769087827f7b86a67b8183",
		"0xb7d9930658124b685bf2bcbd47aea22541c0c5d0",
		"0x99a64c829f5a4c5afbc0a4ea66af2fde060b4ee6",
		"0x5054afce04f7e1b8dcbb388542b4eba7e140d9c8",
		"0x1b29d583468302df6431571d38273b970c3617dc",
		"0x3dc69c6d5ce802a43ed363628d76beebf50b51f8",
		"0xb9953e7213c9529f01de7bb5088eb6f77cb5605b",
		"0xf60b8609a8324fa2f20175091bf57eb81f9c0deb",
		"0xf07303b8a84968fdec7904c9741c2d54a4b40579",
		"0x44b8b9105521616e90b19c6f10e0323513bc4fea",
		"0xecbc2130bbf9336c22e5130984e21a0fb56334e8",
		"0xed6862dcda5acebab0eda74eacddc6d4f8b40f31",
		"0x80eb5e105f5ffd16c0a3ef0b647e89f1ebeb3e72",
		"0xe7c0a7ea3099b868e4cd416bc41cea95929ccdf1",
		"0x5b39ca60ae3a3f74e4dd47049473482cf3145461",
		"0x242fdc1c4d04e294c7293790a177b2b2cebf1fef",
		"0x487274989a7e160ffc67a693f79d1fd09c524b92",
		"0x870021e24661347469a9b7e13f35c3b4e7e37357",
		"0x4ce7c5fa93f682eb880b8dab8517b1a45e49c662",
		"0x225ac4ee12c29db337a0bf7367d2a78291392648",
		"0x54bf50a802423235915f44979c03d68d7ed3a147",
		"0xef55b38f55ed2add70ce0d441598e9f8dbb27285",
		"0x094b59b08c7495b4eada733df3a3a095047859d4",
		"0x2f639e3629970dca3348628d86d2030726d53ccf",
		"0x1e2beefabfd14cb0bb92f5c3c2515a5616481872",
		"0xb81bcc9e4a53bf504df2567afa86633277e9ec98",
		"0x869ee88333c26633c06747bbcee9bfc1fcc29989",
		"0xaf758dab0efcd9b390013dbca01f61121c5c7e21",
		"0x2104d5b752ae7d26ed60ed12d2cba63cffcb981e",
		"0x45ae3870bdba9d754515ee912f0888b7d6e0a20b",
		"0xeb95f9470258df6a9d50dc644003869cb77dca03",
		"0xf588736008ca9084c687993f543435e0e15a2852",
		"0x9a3e8cb939b9ea72f18079d0a3639ce380b2cd31",
	}
)

func accumulateRewardsGen(config *params.ChainConfig, state *state.StateDB, header *types.Header, uncles []*types.Header) {
	G, T, _, E := computerRewardBase(header)

	uncleReward := new(big.Int).Set(T)

	minerReward := new(big.Int).Set(G)

	ecoReward := new(big.Int).Set(E)

	rcount := new(big.Int)
	r := new(big.Int)
	rcd := big.NewInt(0)

	for _, uncle := range uncles {
		state.AddBalance(uncle.Coinbase, uncleReward)

		rcount.Add(rcount, uncleReward)
	}

	r.Div(rcount, big6)

	reward := new(big.Int).Add(minerReward, r)

	state.AddBalance(header.Coinbase, reward)

	for _, cdaddr := range CDAddress {
		state.AddBalance(common.HexToAddress(cdaddr), ecoReward)
		rcd.Add(rcd, ecoReward)
	}

	r1 := new(big.Int).Set(reward)
	r1.Add(r1, rcd)
	r1.Add(r1, rcount)

	totalReward := new(big.Int).Set(header.Rewards)

	r1.Add(r1, totalReward)

	header.Rewards.Set(r1)
}

var blockFiveYearNumber = [...]*big.Int{big.NewInt(3153600), big.NewInt(9460800), big.NewInt(22075200), big.NewInt(47304000), big.NewInt(97761600), big.NewInt(198676800), big.NewInt(400507200)}

func computerRewardBase(header *types.Header) (g, t, l, e *big.Int) {
	gReward := big.NewInt(0)
	tReward := big.NewInt(0)
	lReward := big.NewInt(0)
	eReward := big.NewInt(0)
	totalReward := big.NewInt(0)
	totalReward.Mul(TotalCoin, big.NewInt(1e+18)) //总发现量
	if totalReward.Cmp(header.Rewards) <= 0 {
		return gReward, tReward, lReward, eReward
	}

	blockReward := GenBlockReward
	uncleReward := GenBlockUncleReward

	ecoReward := GenBlockEcoReward
	Greward := new(big.Int).Set(blockReward)
	Treward := new(big.Int).Set(uncleReward)
	Ereward := new(big.Int).Set(ecoReward)
	nums := new(big.Int).Set(header.Number)

	if nums.Cmp(blockFiveYearNumber[0]) <= 0 {
		gReward = new(big.Int).Set(Greward)
		tReward = new(big.Int).Set(Treward)
		eReward = new(big.Int).Set(Ereward)
	} else if nums.Cmp(blockFiveYearNumber[0]) > 0 && nums.Cmp(blockFiveYearNumber[1]) <= 0 {
		gReward.Rsh(Greward, uint(1))
		tReward.Rsh(Treward, uint(1))
		eReward.Rsh(Ereward, uint(1))
	} else if nums.Cmp(blockFiveYearNumber[1]) > 0 && nums.Cmp(blockFiveYearNumber[2]) <= 0 {
		gReward.Rsh(Greward, uint(2))
		tReward.Rsh(Treward, uint(2))
		eReward.Rsh(Ereward, uint(2))
	} else if nums.Cmp(blockFiveYearNumber[2]) > 0 && nums.Cmp(blockFiveYearNumber[3]) <= 0 {
		gReward.Rsh(Greward, uint(3))
		tReward.Rsh(Treward, uint(3))
		eReward.Rsh(Ereward, uint(3))
	} else if nums.Cmp(blockFiveYearNumber[3]) > 0 && nums.Cmp(blockFiveYearNumber[4]) <= 0 {
		gReward.Rsh(Greward, uint(4))
		tReward.Rsh(Treward, uint(4))
		eReward.Rsh(Ereward, uint(4))
	} else if nums.Cmp(blockFiveYearNumber[4]) > 0 && nums.Cmp(blockFiveYearNumber[5]) <= 0 {
		gReward.Rsh(Greward, uint(5))
		tReward.Rsh(Treward, uint(5))
		eReward.Rsh(Ereward, uint(5))
	} else if nums.Cmp(blockFiveYearNumber[5]) > 0 && nums.Cmp(blockFiveYearNumber[6]) <= 0 {
		gReward.Rsh(Greward, uint(6))
		tReward.Rsh(Treward, uint(6))
		eReward.Rsh(Ereward, uint(6))
	} else if nums.Cmp(blockFiveYearNumber[6]) > 0 {
		gReward.Rsh(Greward, uint(7))
		tReward.Rsh(Treward, uint(7))
		eReward.Rsh(Ereward, uint(7))
	}
	return gReward, tReward, lReward, eReward
}
