#!/usr/bin/env python3
#
# Copyright (C) 2020-2024 VyOS maintainers and contributors
#
# This program is free software; you can redistribute it and/or modify
# it under the terms of the GNU General Public License version 2 or later as
# published by the Free Software Foundation.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.

from sys import exit
from jmespath import search
from json import loads

from vyos.config import Config
from vyos.configdict import node_changed
from vyos.configverify import verify_route_map
from vyos.firewall import conntrack_required
from vyos.frrender import FRRender
from vyos.frrender import get_frrender_dict
from vyos.ifconfig import Interface
from vyos.template import render
from vyos.utils.dict import dict_search
from vyos.utils.network import get_vrf_tableid
from vyos.utils.network import get_vrf_members
from vyos.utils.network import interface_exists
from vyos.utils.process import call
from vyos.utils.process import cmd
from vyos.utils.process import is_systemd_service_running
from vyos.utils.process import popen
from vyos.utils.system import sysctl_write
from vyos import ConfigError
from vyos import airbag
airbag.enable()

config_file = '/etc/iproute2/rt_tables.d/vyos-vrf.conf'
k_mod = ['vrf']

nftables_table = 'inet vrf_zones'
nftables_rules = {
    'vrf_zones_ct_in': 'counter ct original zone set iifname map @ct_iface_map',
    'vrf_zones_ct_out': 'counter ct original zone set oifname map @ct_iface_map'
}

def has_rule(af : str, priority : int, table : str=None):
    """
    Check if a given ip rule exists
    $ ip --json -4 rule show
    [{'l3mdev': None, 'priority': 1000, 'src': 'all'},
    {'action': 'unreachable', 'l3mdev': None, 'priority': 2000, 'src': 'all'},
    {'priority': 32765, 'src': 'all', 'table': 'local'},
    {'priority': 32766, 'src': 'all', 'table': 'main'},
    {'priority': 32767, 'src': 'all', 'table': 'default'}]
    """
    if af not in ['-4', '-6']:
        raise ValueError()
    command = f'ip --detail --json {af} rule show'
    for tmp in loads(cmd(command)):
        if 'priority' in tmp and 'table' in tmp:
            if tmp['priority'] == priority and tmp['table'] == table:
                return True
        elif 'priority' in tmp and table in tmp:
            # l3mdev table has a different layout
            if tmp['priority'] == priority:
                return True
    return False

def is_nft_vrf_zone_rule_setup() -> bool:
    """
    Check if an nftables connection tracking rule already exists
    """
    tmp = loads(cmd('sudo nft -j list table inet vrf_zones'))
    num_rules = len(search("nftables[].rule[].chain", tmp))
    return bool(num_rules)

def vrf_interfaces(c, match):
    matched = []
    old_level = c.get_level()
    c.set_level(['interfaces'])
    section = c.get_config_dict([], get_first_key=True)
    for type in section:
        interfaces = section[type]
        for name in interfaces:
            interface = interfaces[name]
            if 'vrf' in interface:
                v = interface.get('vrf', '')
                if v == match:
                    matched.append(name)

    c.set_level(old_level)
    return matched

def vrf_routing(c, match):
    matched = []
    old_level = c.get_level()
    c.set_level(['protocols', 'vrf'])
    if match in c.list_nodes([]):
        matched.append(match)

    c.set_level(old_level)
    return matched

def get_config(config=None):
    if config:
        conf = config
    else:
        conf = Config()

    base = ['vrf']
    vrf = conf.get_config_dict(base, key_mangling=('-', '_'),
                               no_tag_node_value_mangle=True, get_first_key=True)

    # determine which VRF has been removed
    for name in node_changed(conf, base + ['name']):
        if 'vrf_remove' not in vrf:
            vrf.update({'vrf_remove' : {}})

        vrf['vrf_remove'][name] = {}
        # get VRF bound interfaces
        interfaces = vrf_interfaces(conf, name)
        if interfaces: vrf['vrf_remove'][name]['interface'] = interfaces
        # get VRF bound routing instances
        routes = vrf_routing(conf, name)
        if routes: vrf['vrf_remove'][name]['route'] = routes

    if 'name' in vrf:
        vrf['conntrack'] = conntrack_required(conf)

    # We need to merge the FRR rendering dict into the VRF dict
    # this is required to get the route-map information to FRR
    vrf.update({'frr_dict' : get_frrender_dict(conf)})
    return vrf

def verify(vrf):
    # ensure VRF is not assigned to any interface
    if 'vrf_remove' in vrf:
        for name, config in vrf['vrf_remove'].items():
            if 'interface' in config:
                raise ConfigError(f'Can not remove VRF "{name}", it still has '\
                                  f'member interfaces!')
            if 'route' in config:
                raise ConfigError(f'Can not remove VRF "{name}", it still has '\
                                  f'static routes installed!')

    if 'name' in vrf:
        reserved_names = ['add', 'all', 'broadcast', 'default', 'delete', 'dev',
                          'down', 'get', 'inet', 'link', 'mtu', 'type', 'up', 'vrf']

        table_ids = []
        vnis = []
        for name, vrf_config in vrf['name'].items():
            # Reserved VRF names
            if name in reserved_names:
                raise ConfigError(f'VRF name "{name}" is reserved and connot be used!')

            # table id is mandatory
            if 'table' not in vrf_config:
                raise ConfigError(f'VRF "{name}" table id is mandatory!')

            # routing table id can't be changed - OS restriction
            if interface_exists(name):
                tmp = get_vrf_tableid(name)
                if tmp and tmp != int(vrf_config['table']):
                    raise ConfigError(f'VRF "{name}" table id modification not possible!')

            # VRF routing table ID must be unique on the system
            if 'table' in vrf_config and vrf_config['table'] in table_ids:
                raise ConfigError(f'VRF "{name}" table id is not unique!')
            table_ids.append(vrf_config['table'])

            # VRF VNIs must be unique on the system
            if 'vni' in vrf_config:
                vni = vrf_config['vni']
                if vni in vnis:
                    raise ConfigError(f'VRF "{name}" VNI "{vni}" is not unique!')
                vnis.append(vni)

            tmp = dict_search('ip.protocol', vrf_config)
            if tmp != None:
                for protocol, protocol_options in tmp.items():
                    if 'route_map' in protocol_options:
                        verify_route_map(protocol_options['route_map'], vrf['frr_dict'])

            tmp = dict_search('ipv6.protocol', vrf_config)
            if tmp != None:
                for protocol, protocol_options in tmp.items():
                    if 'route_map' in protocol_options:
                        verify_route_map(protocol_options['route_map'], vrf['frr_dict'])

    return None


def generate(vrf):
    # Render iproute2 VR helper names
    render(config_file, 'iproute2/vrf.conf.j2', vrf)

    if 'frr_dict' in vrf and not is_systemd_service_running('vyos-configd.service'):
        FRRender().generate(vrf['frr_dict'])

    return None

def apply(vrf):
    # Documentation
    #
    # - https://github.com/torvalds/linux/blob/master/Documentation/networking/vrf.txt
    # - https://github.com/Mellanox/mlxsw/wiki/Virtual-Routing-and-Forwarding-(VRF)
    # - https://github.com/Mellanox/mlxsw/wiki/L3-Tunneling
    # - https://netdevconf.info/1.1/proceedings/slides/ahern-vrf-tutorial.pdf
    # - https://netdevconf.info/1.2/slides/oct6/02_ahern_what_is_l3mdev_slides.pdf

    # set the default VRF global behaviour
    bind_all = '0'
    if 'bind_to_all' in vrf:
        bind_all = '1'
    sysctl_write('net.ipv4.tcp_l3mdev_accept', bind_all)
    sysctl_write('net.ipv4.udp_l3mdev_accept', bind_all)

    for tmp in (dict_search('vrf_remove', vrf) or []):
        if interface_exists(tmp):
            # T5492: deleting a VRF instance may leafe processes running
            # (e.g. dhclient) as there is a depedency ordering issue in the CLI.
            # We need to ensure that we stop the dhclient processes first so
            # a proper DHCLP RELEASE message is sent
            for interface in get_vrf_members(tmp):
                vrf_iface = Interface(interface)
                vrf_iface.set_dhcp(False)
                vrf_iface.set_dhcpv6(False)

            # Remove nftables conntrack zone map item
            nft_del_element = f'delete element inet vrf_zones ct_iface_map {{ "{tmp}" }}'
            # Check if deleting is possible first to avoid raising errors
            _, err = popen(f'nft --check {nft_del_element}')
            if not err:
                # Remove map element
                cmd(f'nft {nft_del_element}')

            # Delete the VRF Kernel interface
            call(f'ip link delete dev {tmp}')

    if 'name' in vrf:
        # Linux routing uses rules to find tables - routing targets are then
        # looked up in those tables. If the lookup got a matching route, the
        # process ends.
        #
        # TL;DR; first table with a matching entry wins!
        #
        # You can see your routing table lookup rules using "ip rule", sadly the
        # local lookup is hit before any VRF lookup. Pinging an addresses from the
        # VRF will usually find a hit in the local table, and never reach the VRF
        # routing table - this is usually not what you want. Thus we will
        # re-arrange the tables and move the local lookup further down once VRFs
        # are enabled.
        #
        # Thanks to https://stbuehler.de/blog/article/2020/02/29/using_vrf__virtual_routing_and_forwarding__on_linux.html

        for afi in ['-4', '-6']:
            # move lookup local to pref 32765 (from 0)
            if not has_rule(afi, 32765, 'local'):
                call(f'ip {afi} rule add pref 32765 table local')
            if has_rule(afi, 0, 'local'):
                call(f'ip {afi} rule del pref 0')
            # make sure that in VRFs after failed lookup in the VRF specific table
            # nothing else is reached
            if not has_rule(afi, 1000, 'l3mdev'):
                # this should be added by the kernel when a VRF is created
                # add it here for completeness
                call(f'ip {afi} rule add pref 1000 l3mdev protocol kernel')

            # add another rule with an unreachable target which only triggers in VRF context
            # if a route could not be reached
            if not has_rule(afi, 2000, 'l3mdev'):
                call(f'ip {afi} rule add pref 2000 l3mdev unreachable')

        nft_vrf_zone_rule_setup = False
        for name, config in vrf['name'].items():
            table = config['table']
            if not interface_exists(name):
                # For each VRF apart from your default context create a VRF
                # interface with a separate routing table
                call(f'ip link add {name} type vrf table {table}')

            # set VRF description for e.g. SNMP monitoring
            vrf_if = Interface(name)
            # We also should add proper loopback IP addresses to the newly added
            # VRF for services bound to the loopback address (SNMP, NTP)
            vrf_if.add_addr('127.0.0.1/8')
            vrf_if.add_addr('::1/128')
            # add VRF description if available
            vrf_if.set_alias(config.get('description', ''))

            # Enable/Disable IPv4 forwarding
            tmp = dict_search('ip.disable_forwarding', config)
            value = '0' if (tmp != None) else '1'
            vrf_if.set_ipv4_forwarding(value)
            # Enable/Disable IPv6 forwarding
            tmp = dict_search('ipv6.disable_forwarding', config)
            value = '0' if (tmp != None) else '1'
            vrf_if.set_ipv6_forwarding(value)

            # Enable/Disable of an interface must always be done at the end of the
            # derived class to make use of the ref-counting set_admin_state()
            # function. We will only enable the interface if 'up' was called as
            # often as 'down'. This is required by some interface implementations
            # as certain parameters can only be changed when the interface is
            # in admin-down state. This ensures the link does not flap during
            # reconfiguration.
            state = 'down' if 'disable' in config else 'up'
            vrf_if.set_admin_state(state)
            # Add nftables conntrack zone map item
            nft_add_element = f'add element inet vrf_zones ct_iface_map {{ "{name}" : {table} }}'
            cmd(f'nft {nft_add_element}')

        # Only call into nftables as long as there is nothing setup to avoid wasting
        # CPU time and thus lenghten the commit process
        if not nft_vrf_zone_rule_setup:
            nft_vrf_zone_rule_setup = is_nft_vrf_zone_rule_setup()
        # Install nftables conntrack rules only once
        if vrf['conntrack'] and not nft_vrf_zone_rule_setup:
            for chain, rule in nftables_rules.items():
                cmd(f'nft add rule inet vrf_zones {chain} {rule}')

    if 'name' not in vrf or not vrf['conntrack']:
        for chain, rule in nftables_rules.items():
            cmd(f'nft flush chain inet vrf_zones {chain}')

    # Return default ip rule values
    if 'name' not in vrf:
        for afi in ['-4', '-6']:
            # move lookup local to pref 0 (from 32765)
            if not has_rule(afi, 0, 'local'):
                call(f'ip {afi} rule add pref 0 from all lookup local')
            if has_rule(afi, 32765, 'local'):
                call(f'ip {afi} rule del pref 32765 table local')

            if has_rule(afi, 1000, 'l3mdev'):
                call(f'ip {afi} rule del pref 1000 l3mdev protocol kernel')
            if has_rule(afi, 2000, 'l3mdev'):
                call(f'ip {afi} rule del pref 2000 l3mdev unreachable')

    if 'frr_dict' in vrf and not is_systemd_service_running('vyos-configd.service'):
        FRRender().apply()

    return None

if __name__ == '__main__':
    try:
        c = get_config()
        verify(c)
        generate(c)
        apply(c)
    except ConfigError as e:
        print(e)
        exit(1)
