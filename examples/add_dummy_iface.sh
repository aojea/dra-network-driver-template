#!/bin/bash

# Create a dummy interface for testing
# Load the script on one of the nodes and execute
# First argument is the name of the interface
# Second argument the ip address to be assigned

# Note dummy interfaces seems to be destroyed when
# the namespace associated disappears.

ifname=${1:-dummy0}
address=${2:-169.254.169.13/32}

ip link add ${ifname} type dummy
ip link set up dev ${ifname}
ip addr add ${address} dev ${ifname}

