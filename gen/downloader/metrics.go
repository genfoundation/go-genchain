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

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/genchain/go-genchain/metrics"
)

var (
	headerInMeter      = metrics.NewRegisteredMeter("gen/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("gen/downloader/headers/req", nil)
	headerDropMeter    = metrics.NewRegisteredMeter("gen/downloader/headers/drop", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("gen/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("gen/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("gen/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("gen/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("gen/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("gen/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("gen/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("gen/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("gen/downloader/receipts/timeout", nil)

	stateInMeter   = metrics.NewRegisteredMeter("gen/downloader/states/in", nil)
	stateDropMeter = metrics.NewRegisteredMeter("gen/downloader/states/drop", nil)
)
