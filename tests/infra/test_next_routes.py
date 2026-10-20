"""Narrow nginx transformation and packaging guards; no deployment processes."""
import importlib.util
import pathlib
import sys
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'infra/accord/next'))
import render as next_render

spec = importlib.util.spec_from_file_location('public_render', ROOT / 'infra/accord/public/render_public.py')
public = importlib.util.module_from_spec(spec)
spec.loader.exec_module(public)


class NextRoutes(unittest.TestCase):
    def setUp(self):
        self.before = public.render('1.1.1.1', 'tls', web=True)
        self.after = next_render.render(self.before, 7893)

    def test_only_root_replaced_and_ui_routes_added(self):
        start = self.before.index('    location = / {')
        end = self.before.index('\n    }', start) + len('\n    }')
        self.assertTrue(self.after.startswith(next_render.ASSET_ZONE + self.before[:start]))
        self.assertTrue(self.after.endswith(self.before[end:]))
        self.assertIn('location = / { return 302 /app; }', self.after)
        self.assertEqual(self.after.count('proxy_pass http://127.0.0.1:7893;'), 5)
        self.assertIn('location ^~ /_next/static/', self.after)
        self.assertNotIn('location ^~ /_next/ {', self.after)
        self.assertNotIn('/_next/image', self.after)

    def test_go_upload_and_native_bytes_preserved(self):
        native = '    location ~ ^/(api/v1'
        self.assertEqual(self.after[self.after.index(native):self.after.index('    # Accord Next')],
                         self.before[self.before.index(native):self.before.index('    location = / {')])
        marker = '    location /web/ {'
        self.assertEqual(self.after[self.after.index(marker):], self.before[self.before.index(marker):])
        self.assertIn('client_max_body_size 52494336;', self.after)

    def test_assets_have_separate_bounded_rate_and_app_preserves_parent_rate(self):
        self.assertEqual(self.after.count(next_render.ASSET_ZONE), 1)
        self.assertEqual(self.after.count('limit_req zone=accord_next_assets burst=100 nodelay;'), 3)
        for route in ('= /app', '^~ /app/'):
            start = self.after.index('    location ' + route + ' {')
            end = self.after.index('\n    }', start)
            self.assertNotIn('limit_req ', self.after[start:end])
        for route in ('^~ /_next/static/', '^~ /fonts/', '= /PAPERCLIP-LICENSE.txt'):
            start = self.after.index('    location ' + route + ' {')
            end = self.after.index('\n    }', start)
            self.assertIn('limit_req zone=accord_next_assets burst=100 nodelay;', self.after[start:end])
        self.assertEqual(self.after.count('limit_req zone=accord_public_rate burst=40 nodelay;'),
                         self.before.count('limit_req zone=accord_public_rate burst=40 nodelay;'))
        self.assertEqual(self.after.count('limit_conn accord_public_connections 32;'),
                         self.before.count('limit_conn accord_public_connections 32;'))
        # The entire plaintext/ACME block and all old HTTP-level definitions
        # remain byte-identical below the added independent zone declaration.
        tls = 'map $http_host $accord_https_authority'
        self.assertEqual(self.after[len(next_render.ASSET_ZONE):self.after.index(tls)],
                         self.before[:self.before.index(tls)])

    def test_repeated_unknown_and_unsafe_ports_rejected(self):
        for value in ('7892', '07893', '80', '65536', '7893; }'):
            with self.subTest(value=value), self.assertRaises(ValueError):
                next_render.render(self.before, value)
        with self.assertRaises(ValueError):
            next_render.render(self.after, 7893)
        with self.assertRaises(ValueError):
            next_render.render(self.before.replace('location = / {', 'location = /other {'), 7893)
        with self.assertRaises(ValueError):
            next_render.render(self.before.replace('52494336', '100m'), 7893)

    def test_next_header_sanitization_does_not_rewrite_origin(self):
        next_part = self.after.split('    # Accord Next')[1].split('    location /web/')[0]
        self.assertNotIn('proxy_set_header Origin', next_part)
        self.assertNotIn('proxy_set_header Authorization', next_part)
        self.assertIn('proxy_set_header Host $http_host;', next_part)
        self.assertIn('proxy_set_header X-Accord-Client-IP "";', next_part)


if __name__ == '__main__':
    unittest.main()
