module tfgrid

import json

// wg network reservation (znet)

pub struct Znet {
pub mut:
	// unique nr for each network chosen, this identified private networks as connected to a container or vm or ...
	// corresponds to the 2nd number of a class B ipv4 address
	// is a class C of a chosen class B
	
	// IPV4 subnet for this network resource
	// this must be a valid subnet of the entire network ip range.
	// for example 10.1.1.0/24
	subnet   string  
	// IP range of the network, must be an IPv4 /16
	// for example a 10.1.0.0/16
	ip_range string 
	// wireguard private key, curve25519
	// TODO: is this in libsodium 
	wireguard_private_key string //TODO: what is format
	//>1024?
	wireguard_listen_port u16
	peers                 []Peer
}

pub fn (mut n Znet) challenge() string {
	mut out := ''
	out += n.ip_range
	out += n.subnet
	out += n.wireguard_private_key
	out += n.wireguard_listen_port.str()
	for mut p in n.peers {
		out += p.challenge()
	}

	return out
}

// is a remote wireguard client which can connect to this node
pub struct Peer {
pub mut:
	
	subnet string // IPV4 subnet of the network resource of the peer

	
	wireguard_public_key string    // WGPublicKey of the peer (driven from its private key)

	//is ipv4 or ipv6 address from a wireguard client who connects
	//this should be the node's subnet and the wireguard routing ip that should start with `100.64`
	//then the 2nd and 3rd part of the node's subnet
	//e.g. ["10.20.2.0/24", "100.64.20.2/32"]
	allowed_ips          []string 
	// Entrypoint of the peer; ipv4 or ipv6,
	// can be empty, one of the 2 need to be filled in though
	//e.g. [2a10:b600:0:9:225:90ff:fe82:7130]:7777
	endpoint string 
}



pub fn (mut p Peer) challenge() string {
	mut out := ''
	out += p.wireguard_public_key
	out += p.endpoint
	out += p.subnet

	for ip in p.allowed_ips {
		out += ip
	}
	return out
}

pub fn (z Znet) to_workload(args WorkloadArgs) Workload {
	return Workload{
		version: args.version or { 0 }
		name: args.name
		type_: workload_types.network
		data: json.encode(z)
		metadata: args.metadata or { '' }
		description: args.description or { '' }
		result: args.result or { WorkloadResult{} }
	}
}
