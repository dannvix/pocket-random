#!/usr/bin/env python
# encoding: utf-8
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
from datetime import datetime

import requests

# blessings
try:
    from blessings import Terminal
except ImportError:
    # fallback
    class Terminal(object):
        def __getattr__(self, name):
            if name == 'width': return 100
            elif name == 'height': return 24
            def _missing(*args, **kwargs):
                return ''.join(args) or None
            return _missing


# human-friendly date
# ref. http://stackoverflow.com/a/1551394
def pretty_date(timestamp=None):
    now = datetime.now()
    if type(timestamp) is int:
        diff = now - datetime.fromtimestamp(timestamp)
    elif isinstance(timestamp, datetime):
        diff = now - timestamp
    elif not None:
        diff = now - now

    day_diff = diff.days
    if day_diff < 0:
        return 'from the future!'
    elif day_diff == 0:
        second_diff = diff.seconds
        if second_diff < 60:
            return 'few seconds ago'
        if second_diff < 120:
            return 'a minute ago'
        if second_diff < 3600:
            return str(second_diff / 60) + ' minutes ago'
        if second_diff < 7200:
            return 'an hour ago'
        if second_diff < 86400:
            return str(second_diff / 3600) + ' hours ago'
    elif day_diff == 1:
        return "yesterday"
    elif day_diff < 7:
        return str(day_diff) + ' days ago'
    elif day_diff < 31:
        return str(day_diff / 7) + ' weeks ago'
    elif day_diff < 365:
        return str(day_diff / 30) + ' months ago'
    return str(day_diff / 365) + ' years ago'


# truncate string
def truncate(string, width):
    if len(string) > width:
      return u'{truncated}…'.format(truncated=string[:width-1])
    return string


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
    # colored terminal output
    t = Terminal()

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
        print(t.yellow('Hello {username}!'.format(username=cfg.username)))
        sys.stdout.write('Retrieving items from Pocket... '); sys.stdout.flush()
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

        random.shuffle(items)
        while items:
            picked_item = items.pop(0)
            item_id = picked_item.get('item_id')
            item_title = picked_item.get('resolved_title')
            item_url = picked_item.get('resolved_url')
            item_timestamp = int(picked_item.get('time_added', 0))

            id_field = t.yellow(u'[#{id}]'.format(id=item_id))
            title_field = t.white(u'"{title}"'.format(title=item_title))
            url_field = t.green(u'{url}'.format(url=truncate(item_url, t.width)))
            date_field = t.blue(u'Added at {date}'.format(date=pretty_date(item_timestamp)))

            print(u'')
            print(u'{id} {title}'.format(id=id_field, title=title_field))
            print(u'{url}'.format(url=url_field))
            print(u'{date}'.format(date=date_field))

            while True:
                answer = raw_input('Action> open(o), archive(a), next(n), quit(q) ? ').lower()
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
                            'item_id': item_id}])}
                    response = requests.get(request_url, params=request_data)
                    if response.status_code != requests.codes.ok:
                        raise Exception('archive item error, status_code=[{status_code}], X-Error-Code=[{xerror}]'.format(
                            status_code=response.status_code, xerror=response.headers.get('X-Error-Code')))
                    else:
                        print('Item #{item_id} archived :-)'.format(item_id=item_id))
                    break
