module main

import freeflowuniverse.crystallib.rpcwebsocket { RpcWsClient }
import flag
import log
import os
import btc

const (
	default_server_address = 'http://127.0.0.1:8080'
)

fn execute_rpcs(mut client RpcWsClient, mut logger log.Logger, host string, user string, password string) ! {
	mut btc_client := btc.new(mut client)

	btc_client.load(host: host, user: user, pass: password)!

	amount_blocks := btc_client.get_block_count()!
	logger.info("Block count: ${amount_blocks}")
}

fn main() {
	mut fp := flag.new_flag_parser(os.args)
	fp.application('Welcome to the web3_proxy client. The web3_proxy client allows you to execute all remote procedure calls that the web3_proxy server can handle.')
	fp.limit_free_args(0, 0)!
	fp.description('')
	fp.skip_executable()
	address := fp.string('address', `a`, '${default_server_address}', 'The address of the web3_proxy server to connect to.')
	password := fp.string('pass', `p`, '', 'The password to use to connect to the bitcoin node.')
	host := fp.string('host', `h`, '', 'The address of the bitcoin node to connect to.')
	user := fp.string('user', `u`, '', 'The user to use to connect to the bitcoin node.')
	debug_log := fp.bool('debug', 0, false, 'By setting this flag the client will print debug logs too.')

	_ := fp.finalize() or {
		eprintln(err)
		println(fp.usage())
		exit(1)
	}

	mut logger := log.Logger(&log.Log{
		level: if debug_log { .debug } else { .info }
	})

	mut myclient := rpcwebsocket.new_rpcwsclient(address, &logger) or {
		logger.error('Failed creating rpc websocket client: ${err}')
		exit(1)
	}

	_ := spawn myclient.run()

	execute_rpcs(mut myclient, mut logger, host, user, password) or {
		logger.error('Failed executing calls: ${err}')
		exit(1)
	}
}
