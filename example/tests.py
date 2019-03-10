# -*- coding: utf-8 -*-

import json
import os
import unittest
import urllib2

TEST_APP_PORT = int(os.getenv('TEST_APP_PORT', 9001))
TEST_CHAMELEON_PORT = int(os.getenv('TEST_CHAMELEON_PORT', 6005))


def get_name(code):
    url = 'http://localhost:{}/{}'.format(TEST_APP_PORT, code)
    try:
        resp = urllib2.urlopen(url)
    except urllib2.HTTPError as exc:
        resp = exc
    return resp.read()


def preseed(url, method):
    payload = json.dumps({
        'Request': {
            'URL': url,
            'Method': method,
            'Body': '',
        },
        'Response': {
            'StatusCode': 942,
            'Body': '{"key": "value"}',
            'Headers': {
                'Content-Type': 'application/json',
            }
        },
    })
    req = urllib2.Request('http://localhost:{}/_seed'.format(TEST_CHAMELEON_PORT), payload, {'Content-type': 'application/json'})
    req.get_method = lambda: 'POST'
    resp = urllib2.urlopen(req)

    return resp


class MyTest(unittest.TestCase):

    def test_200_returns_ok(self):
        self.assertEqual('OK', get_name(200))

    def test_418_returns_teapot(self):
        self.assertEqual("I'M A TEAPOT", get_name(418))

    def test_500_internal_server_error(self):
        self.assertEqual('INTERNAL SERVER ERROR', get_name(500))

    def test_content_type_is_text_plain(self):
        url = 'http://localhost:{}/200'.format(TEST_APP_PORT)
        resp = urllib2.urlopen(url)
        self.assertEqual('text/plain', resp.headers['content-type'])

    def test_post_returns_post_body(self):
        url = 'http://localhost:{}/post'.format(TEST_APP_PORT)
        req = urllib2.Request(url, json.dumps({'foo': 'bar'}), {'Content-type': 'application/json'})
        req.get_method = lambda: 'POST'
        resp = urllib2.urlopen(req)
        parsed = json.loads(resp.read())
        self.assertEqual({'foo': 'bar'}, parsed['json'])

        # now with hashed
        url = 'http://localhost:{}/post_with_body'.format(TEST_APP_PORT)
        req = urllib2.Request(url, json.dumps({'post': 'body'}), {'Content-type': 'application/json'})
        req.get_method = lambda: 'HASHED'
        resp = urllib2.urlopen(req)
        parsed = json.loads(resp.read())
        self.assertEqual({'foo': 'bar'}, parsed['json'])

    def test_patch_returns_body(self):
        url = 'http://localhost:{}/patch'.format(TEST_APP_PORT)
        req = urllib2.Request(url, json.dumps({'hi': 'hello'}), {'Content-type': 'application/json'})
        req.get_method = lambda: 'PATCH'
        resp = urllib2.urlopen(req)
        parsed = json.loads(resp.read())
        self.assertEqual({'hi': 'hello'}, parsed['json'])

    def test_put_returns_body(self):
        url = 'http://localhost:{}/put'.format(TEST_APP_PORT)
        req = urllib2.Request(url, json.dumps({'spam': 'eggs'}), {'Content-type': 'application/json'})
        req.get_method = lambda: 'PUT'
        resp = urllib2.urlopen(req)
        parsed = json.loads(resp.read())
        self.assertEqual({'spam': 'eggs'}, parsed['json'])

    def test_delete_returns_200(self):
        url = 'http://localhost:{}/delete'.format(TEST_APP_PORT)
        req = urllib2.Request(url)
        req.get_method = lambda: 'DELETE'
        resp = urllib2.urlopen(req)
        self.assertEqual(200, resp.getcode())

    def test_preseed(self):
        resp = preseed('/encoding/utf8', 'GET')  # Preseed this URL and Method with some data
        self.assertIn(resp.getcode(), (200, 201))
        url = 'http://localhost:{}/encoding/utf8'.format(TEST_APP_PORT)
        req = urllib2.Request(url)
        req.get_method = lambda: 'SEEDED'
        try:
            resp = urllib2.urlopen(req)
        except urllib2.HTTPError as exc:
            resp = exc
        self.assertEqual('application/json', resp.headers['content-type'])
        self.assertEqual(942, resp.getcode())
        self.assertEqual({'key': 'value'}, json.loads(resp.read()))

    def test_specify_hash_in_request(self):
        url = 'http://localhost:{}/post'.format(TEST_APP_PORT)
        req = urllib2.Request(url, json.dumps({'foo': 'bar'}), {
            'Content-type': 'application/json', 'chameleon-request-hash': 'foo_bar_hash'})
        req.get_method = lambda: 'REQUESTHASH'
        resp = urllib2.urlopen(req)
        content = resp.read()
        parsed = json.loads(content)
        self.assertEqual({'foo': 'bar'}, parsed['json'])
        self.assertEqual('foo_bar_hash', resp.headers['chameleon-request-hash'])


if __name__ == '__main__':
    unittest.main()
