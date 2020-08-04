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

package params

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main Genchain network.
var MainnetBootnodes = []string{
	// Genchain Foundation Go Bootnodes
	"enode://a737cabb1c73fc8ae5e0db9b1874d7162539bd8ae8fb959226b30fc5f583ed05eb37e4c0d101923e2db70b6ad40e5e01ff17afa503613f285ff4a73a0edcdde5@119.27.170.108:60606",
	// "enode://5c579ba4feee6bceeeea36ab81ed9e151fec24f81c8e4d3144d08dc9f5e94441e3f57be466c1b5cc3d219f1d6a3e003b400c3e131ae72cb2584ea8532e483b2a@150.109.23.164:60606",
	"enode://beba2e70d96f9be576242978d0cfaa5d720041d5353fcd67c3ed303f14c86914e32831ae2173ef5adfa0ef00db152aa639897281311a62a3686b93249c247f58@212.64.51.154:60606",
	"enode://c4c6ae65cf958fb527a077ed27b032b3562facf581b5f1fc64a82304affb7b46ffc5e2dca0f901022ffc0442daf0831544bf0b8520e36b5d7bb33ecdbcf7d495@152.136.233.254:60606",
	//"enode://7256af18c8614b669cc787393fa341833ea0ae2775d9f7ef5d91ebaa9db6cac5a169594c647a61f1394a2d53ac3f4967401f37260ec7961127399109be062362@152.136.233.254:60606", // BJ
	//"enode://09815abff9beb1e060df019beb96ee45a591f3cde19930c6e9b63147083e255d48905298f6f955848d0a9c0b1cebf8b40b9f76995c46b40726ee1be2fb687a36@119.27.170.108:60606",  // ZH-CD
	//"enode://e8e805bbb25a056d367c23ab99b7879c7fb6a8d828b6bc939e7b80030db0d7e337ccbe3030d2dd63a2a3a9f70bc63836c9108e49ce4066936658fdb7ee043ba5@212.64.51.154:60606",   // ZH-SH

}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Ropsten test network.
var TestnetBootnodes = []string{
	// "enode://30b7ab30a01c124a6cceca36863ece12c4f5fa68e3ba9b0b51407ccc002eeed3b3102d20a88f1c1d3c3154e2449317b8ef95090e77b312d5cc39354f86d5d606@52.176.7.10:60606",    // US-Azure ggen
	// "enode://865a63255b3bb68023b6bffd5095118fcc13e79dcf014fe4e47e065c350c7cc72af2e53eff895f11ba1bbb6a2b33271c1116ee870f266618eadfc2e78aa7349c@52.176.100.77:60606",  // US-Azure parity
	// "enode://6332792c4a00e3e4ee0926ed89e0d27ef985424d97b6a45bf0f23e51f0dcb5e66b875777506458aea7af6f9e4ffb69f43f3778ee73c81ed9d34c51c4b16b0b0f@52.232.243.152:60606", // Parity
	// "enode://94c15d1b9e2fe7ce56e458b9a3b672ef11894ddedd0c6f247e0f1d3487f52b66208fb4aeb8179fce6e3a749ea93ed147c37976d67af557508d199d9594c35f09@192.81.208.223:60606", // @gpip
}

// RinkebyBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Rinkeby test network.
var RinkebyBootnodes = []string{
	// "enode://a24ac7c5484ef4ed0c5eb2d36620ba4e4aa13b8c84684e1b4aab0cebea2ae45cb4d375b77eab56516d34bfbd3c1a833fc51296ff084b770b94fb9028c4d25ccf@52.169.42.101:60606", // IE
	// "enode://343149e4feefa15d882d9fe4ac7d88f885bd05ebb735e547f12e12080a9fa07c8014ca6fd7f373123488102fe5e34111f8509cf0b7de3f5b44339c9f25e87cb8@52.3.158.184:60606",  // INFURA
	// "enode://b6b28890b006743680c52e64e0d16db57f28124885595fa03a562be1d2bf0f3a1da297d56b13da25fb992888fd556d4c1a27b1f39d531bde7de1921c90061cc6@159.89.28.211:60606", // AKASHA
}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
var DiscoveryV5Bootnodes = []string{
	// "enode://06051a5573c81934c9554ef2898eb13b33a34b94cf36b202b69fde139ca17a85051979867720d4bdae4323d4943ddf9aeeb6643633aa656e0be843659795007a@35.177.226.168:60606",
	// "enode://0cc5f5ffb5d9098c8b8c62325f3797f56509bff942704687b6530992ac706e2cb946b90a34f1f19548cd3c7baccbcaea354531e5983c7d1bc0dee16ce4b6440b@40.118.3.223:30304",
	// "enode://1c7a64d76c0334b0418c004af2f67c50e36a3be60b5e4790bdac0439d21603469a85fad36f2473c9a80eb043ae60936df905fa28f1ff614c3e5dc34f15dcd2dc@40.118.3.223:30306",
	// "enode://85c85d7143ae8bb96924f2b54f1b3e70d8c4d367af305325d30a61385a432f247d2c75c45c6b4a60335060d072d7f5b35dd1d4c45f76941f62a4f83b6e75daaf@40.118.3.223:30307",
}
