module main

import log
import threefoldtech.tfgrid
import threefoldtech.tfgrid.models
fn main() {
	mut logger := log.Log{
		level: .debug
	}

	mnemonics := tfgrid.get_mnemonics() or {
		logger.error(err.str())
		exit(1)
	}
	chain_network := tfgrid.ChainNetwork.dev // Use your desired network
	mut deployer := tfgrid.new_deployer(mnemonics, chain_network, mut logger)!

	zdb := models.Zdb{
		size: u64(2) * 1024 * 1024
		mode: 'user'
		password: 'pass'
	}

	wl := zdb.to_workload(name: 'mywlname')

	signature_requirement := models.SignatureRequirement{
		weight_required: 1
		requests: [
			models.SignatureRequest{
				twin_id: deployer.twin_id
				weight: 1
			},
		]
	}

	mut deployment := models.new_deployment(
		twin_id: deployer.twin_id
		workloads: [wl]
		signature_requirement: signature_requirement
	)

	node_contract_id := deployer.deploy(33, mut deployment, '', 0)!
	logger.info('node contract created with id ${node_contract_id}')
}
