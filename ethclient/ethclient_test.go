// Copyright 2018The go-genchain Authors
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

package ethclient

import "github.com/genchain/go-genchain"

// Verify that Client implements the genchain interfaces.
var (
	_ = genchain.ChainReader(&Client{})
	_ = genchain.TransactionReader(&Client{})
	_ = genchain.ChainStateReader(&Client{})
	_ = genchain.ChainSyncReader(&Client{})
	_ = genchain.ContractCaller(&Client{})
	_ = genchain.GasEstimator(&Client{})
	_ = genchain.GasPricer(&Client{})
	_ = genchain.LogFilterer(&Client{})
	_ = genchain.PendingStateReader(&Client{})
	// _ = genchain.PendingStateEventer(&Client{})
	_ = genchain.PendingContractCaller(&Client{})
)
