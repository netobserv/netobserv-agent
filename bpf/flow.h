#ifndef __FLOW_H__
#define __FLOW_H__

#define TC_ACT_OK 0
#define TC_ACT_SHOT 2
#define IP_MAX_LEN 16

// TODO : Explore if this can be programmed from go launcher
// Total mem consumption would be (1000*)
#define INGRESS_MAX_ENTRIES 1000
#define EGRESS_MAX_ENTRIES  1000


// Bitmask of flags to be embedded in the 32-bit
// In Future, Other TCP Flags can be added
#define TCP_FIN_FLAG 0x1
#define TCP_RST_FLAG 0x10

typedef __u8 u8;
typedef __u16 u16;
typedef __u32 u32;
typedef __u64 u64;


// L2 data link layer
struct data_link {
    u8 src_mac[ETH_ALEN];
    u8 dst_mac[ETH_ALEN];
} __attribute__((packed));

// L3 network layer
// IPv4 addresses are encoded as IPv6 addresses with prefix ::ffff/96
// as described in https://datatracker.ietf.org/doc/html/rfc4038#section-4.2
struct network {
    struct in6_addr src_ip;
    struct in6_addr dst_ip;
} __attribute__((packed));

// L4 transport layer
struct transport {
    u16 src_port;
    u16 dst_port;
    u8 protocol;
} __attribute__((packed));

// TODO: L5 session layer to bound flows to connections?

// contents in this struct must match byte-by-byte with Go's pkc/flow/Record struct
typedef struct flow_t {
    u16 protocol;
    u8 direction;
    struct data_link data_link;
    struct network network;
    struct transport transport;
} __attribute__((packed)) flow;


typedef struct flow_metrics_t {
	__u32 packets;
	__u64 bytes;
	__u64 flow_start_ts;
    __u64 last_pkt_ts;
	__u32 flags;  // Could be used to indicate certain things
} __attribute__((packed)) flow_metrics;

typedef struct flow_id_t {
    u16 eth_protocol;
    u8 src_mac[ETH_ALEN];
    u8 dst_mac[ETH_ALEN];
    struct in6_addr src_ip;
    struct in6_addr dst_ip;
    u16 src_port;
    u16 dst_port;
    u8 protocol;
} __attribute__((packed)) flow_id_v;

// Flow record is the typical information sent from eBPF to userspace
typedef struct flow_record_t {
	flow flow_key;
	flow_metrics metrics;
} __attribute__((packed)) flow_record;
#endif
