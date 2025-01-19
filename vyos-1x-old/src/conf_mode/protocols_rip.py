#!/usr/bin/env python3
#
# Copyright (C) 2021-2024 VyOS maintainers and contributors
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

from vyos.config import Config
from vyos.configverify import has_frr_protocol_in_dict
from vyos.configverify import verify_common_route_maps
from vyos.configverify import verify_access_list
from vyos.configverify import verify_prefix_list
from vyos.frrender import FRRender
from vyos.frrender import get_frrender_dict
from vyos.utils.dict import dict_search
from vyos.utils.process import is_systemd_service_running
from vyos import ConfigError
from vyos import airbag
airbag.enable()

def get_config(config=None):
    if config:
        conf = config
    else:
        conf = Config()

    return get_frrender_dict(conf)

def verify(config_dict):
    if not has_frr_protocol_in_dict(config_dict, 'rip'):
        return None

    rip = config_dict['rip']
    rip['policy'] = config_dict['policy']

    verify_common_route_maps(rip)

    acl_in = dict_search('distribute_list.access_list.in', rip)
    if acl_in: verify_access_list(acl_in, rip)

    acl_out = dict_search('distribute_list.access_list.out', rip)
    if acl_out: verify_access_list(acl_out, rip)

    prefix_list_in = dict_search('distribute_list.prefix-list.in', rip)
    if prefix_list_in: verify_prefix_list(prefix_list_in, rip)

    prefix_list_out = dict_search('distribute_list.prefix_list.out', rip)
    if prefix_list_out: verify_prefix_list(prefix_list_out, rip)

    if 'interface' in rip:
        for interface, interface_options in rip['interface'].items():
            if 'authentication' in interface_options:
                if {'md5', 'plaintext_password'} <= set(interface_options['authentication']):
                    raise ConfigError('Can not use both md5 and plaintext-password at the same time!')
            if 'split_horizon' in interface_options:
                if {'disable', 'poison_reverse'} <= set(interface_options['split_horizon']):
                    raise ConfigError(f'You can not have "split-horizon poison-reverse" enabled ' \
                                      f'with "split-horizon disable" for "{interface}"!')

def generate(config_dict):
    if config_dict and not is_systemd_service_running('vyos-configd.service'):
        FRRender().generate(config_dict)
    return None

def apply(config_dict):
    if config_dict and not is_systemd_service_running('vyos-configd.service'):
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
