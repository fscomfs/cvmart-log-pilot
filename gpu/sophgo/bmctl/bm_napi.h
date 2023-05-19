#ifndef __BM_DRIVER_VNIC_H_
#define __BM_DRIVER_VNIC_H_


#define ETH_MTU 65536
#define VETH_DEVICE_NAME "veth"

#define GP_REG15_SET 0x198

#define HOST_READY_FLAG 0x01234567
#define DEVICE_READY_FLAG 0x89abcde

#define VETH_TX_QUEUE_PHY 0x0
#define VETH_TX_QUEUE_LEN 0x8
#define VETH_TX_QUEUE_HEAD 0x10
#define VETH_TX_QUEUE_TAIL 0x14

#define VETH_RX_QUEUE_PHY 0x0
#define VETH_RX_QUEUE_LEN 0x8
#define VETH_RX_QUEUE_HEAD 0x10
#define VETH_RX_QUEUE_TAIL 0x14

#define VETH_HANDSHAKE_REG 0x40
#define VETH_IPADDRESS_REG 0x44
#define VETH_A53_STATE_REG 0x48

#define VETH_SHM_START_ADDR 0x0201be80

typedef unsigned char      u8;
typedef unsigned short     u16;
typedef unsigned int       u32;
typedef unsigned long long u64;
typedef unsigned long long dma_addr_t;
typedef signed char s8;
typedef signed short s16;
typedef unsigned int u32;
typedef signed int s32;
typedef signed long long int s64;

#endif
