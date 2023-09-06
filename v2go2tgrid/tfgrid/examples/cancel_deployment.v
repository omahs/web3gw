module main

import log
import threefoldtech.tfgrid

fn main() {
	mut logger := log.Logger(&log.Log{
		level: .debug
	})

	mnemonics := '<YOUR MNEMONICS>'
	chain_network := tfgrid.ChainNetwork.dev // User your desired network
	mut deployer := tfgrid.new_deployer(mnemonics, chain_network)!

	contract_id := u64(37459) // replace with contract id that you want to cancel
	deployer.cancel_contract(contract_id)!

	logger.info('contract ${contract_id} is canceled')
}
