#!/usr/bin/env python
# pocket-random.py -- randomly pick up some items from Pocket

# Copyright (c) 2016 Shao-Chung Chen
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.

import json
import os
import random
import sys
import time
import webbrowser

import requests

# API configurations
API_BASE = 'https://getpocket.com/v3/'

class UserConfig(object):
    def __init__(self):
        self.filename = '.pocketrandom'
        self.pathname = os.path.join(os.path.expanduser('~'), self.filename)
        self.api_key = None
        self.username = None
        self.user_code = None
        self.user_token = None
        self.load_or_create_new()

    def load_or_create_new(self):
        try:
            with open(self.pathname, 'rb') as cfgfile:
                cfg = json.loads(cfgfile.read())
                self.api_key = cfg.get('api_key')
                self.username = cfg.get('username')
                self.user_code = cfg.get('user_code')
                self.user_token = cfg.get('user_token')
        except (IOError, ValueError):
            # config file not existing or invalid, create new/empty
            self.save()

    def save(self):
        with open(self.pathname, 'wb') as cfgfile:
            cfgfile.write(json.dumps({
                'api_key': self.api_key,
                'username': self.username,
                'user_code': self.user_code,
                'user_token': self.user_token}))


if __name__ == '__main__':
    # config
    cfg = UserConfig()

    # API key
    if not cfg.api_key:
        pocket_devsite = 'http://getpocket.com/developer/apps/'
        print('No API key available. Get one on {devapps}'.format(devapps=pocket_devsite))
        webbrowser.open(pocket_devsite)
        cfg.api_key = raw_input('Enter your API key: ').strip()
        cfg.save()

    # OAuth authentication
    if not cfg.user_code and not cfg.user_token:
        redirect_uri = 'https://getpocket.com/connected_accounts'
        request_url = '{api_base}oauth/request'.format(api_base=API_BASE)
        request_data = {
            'consumer_key': cfg.api_key,
            'redirect_uri': redirect_uri}
        request_headers = {
            'X-Accept': 'application/json'}

        response = requests.post(request_url, data=request_data, headers=request_headers)
        if response.status_code != requests.codes.ok:
            raise Exception('request code error, status_code=[{status_code}], X-Error-Code=[{xerror}]'.format(
                status_code=response.status_code, xerror=response.headers.get('X-Error-Code')))

        cfg.user_code = response.json()['code']
        print('OAuth code: {oauth_code}'.format(oauth_code=cfg.user_code))
        authorization_url = 'https://getpocket.com/auth/authorize?request_token={request_token}&redirect_uri={redirect_uri}'.format(
            request_token=cfg.user_code, redirect_uri=redirect_uri)
        print('Please authroize this app on {auth_url}'.format(auth_url=authorization_url))
        webbrowser.open(authorization_url)
        raw_input('Press any key to continue...')
        cfg.save()

    if not cfg.user_token:
        request_url = '{api_base}oauth/authorize'.format(api_base=API_BASE)
        request_data = {
            'consumer_key': cfg.api_key,
            'code': cfg.user_code}
        request_headers = {
            'X-Accept': 'application/json'}
        response = requests.post(request_url, data=request_data, headers=request_headers)
        if response.status_code != requests.codes.ok:
            raise Exception('request authorization error, status_code=[{status_code}], X-Error-Code=[{xerror}]'.format(
                status_code=response.status_code, xerror=response.headers.get('X-Error-Code')))

        cfg.username = response.json()['username']
        cfg.user_token = response.json()['access_token']
        print('Username: {username}'.format(username=cfg.username))
        print('Auth token: {oauth_token}'.format(oauth_token=cfg.user_token))
        cfg.save()

    if cfg.api_key and cfg.user_code and cfg.user_token:
        print('Hello {username}!'.format(username=cfg.username))
        print('')
        print('Retrieving items from Pocket...')
        # FIXME: retrieve items in small batches (API limitation = 5000 items for each requests)
        # FIXME: save the retrieved data if possible
        request_url = '{api_base}get'.format(api_base=API_BASE)
        request_data = {
            'consumer_key': cfg.api_key,
            'access_token': cfg.user_token,
            'detailType': 'simple'}
        response = requests.post(request_url, data=request_data)
        items = response.json()['list'].values()
        item_count = len(items)
        print('{item_count} items retrieved!'.format(item_count=item_count))

        while True:
            picked_item = random.choice(items)
            item_id = picked_item.get('item_id')
            item_title = picked_item.get('resolved_title').encode('utf8')
            item_url = picked_item.get('resolved_url').encode('utf8')
            print('')
            print('Item #{item_id} - "{item_title}"({item_url})'.format(
                item_id=item_id, item_title=item_title, item_url=item_url))

            while True:
                answer = raw_input('Action? open(o), archive(a), next(n), quit(q)? ').lower()
                if answer in ['o', 'open']:
                    webbrowser.open(item_url)
                elif answer in ['n', 'next']:
                    break
                elif answer in ['q', 'quit']:
                    sys.exit(0)
                elif answer in ['a', 'archive']:
                    request_url = '{api_base}send'.format(api_base=API_BASE)
                    request_data = {
                        'consumer_key': cfg.api_key,
                        'access_token': cfg.user_token,
                        'actions': json.dumps([{
                            'action': 'archive',
                            'item_id': item_id,
                            'time': int(time.time())}])}
                    response = requests.get(request_url, params=request_data)
                    if response.status_code != requests.codes.ok:
                        raise Exception('archive item error, status_code=[{status_code}], X-Error-Code=[{xerror}]'.format(
                            status_code=response.status_code, xerror=response.headers.get('X-Error-Code')))
                    else:
                        print('Item #{item_id} archived'.format(item_id=item_id))
                    break
