"""Security contracts for the optional public browser ingress."""
import importlib.util
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location("public_ingress", ROOT / "infra/accord/public/render_public.py")
ingress = importlib.util.module_from_spec(spec)
spec.loader.exec_module(ingress)


class WebProxyTests(unittest.TestCase):
    def test_default_native_ingress_stays_closed_to_web(self):
        value = ingress.render("8.8.8.8", "tls")
        self.assertNotIn("location /web/", value)
        self.assertEqual(value.count("client_max_body_size 64k"), 2)

    def test_web_changes_neither_plaintext_nor_native_body_limit(self):
        value = ingress.render("8.8.8.8", "tls", True)
        plain, tls = value.split("map $http_host $accord_https_authority", 1)
        self.assertNotIn("proxy_pass", plain)
        self.assertNotIn("/web/", plain)
        self.assertIn("client_max_body_size 64k", tls)
        upload = tls.split("location = /web/designs {", 1)[1].split("}", 1)[0]
        self.assertIn("client_max_body_size 52494336", upload)
        self.assertIn("proxy_request_buffering off", upload)
        self.assertIn("client_body_timeout 120s", upload)
        self.assertNotIn("proxy_set_header Origin", value)
        self.assertNotIn("proxy_set_header Authorization", value)

    def test_trusted_ip_is_overwritten_and_metrics_stays_private(self):
        value = ingress.render("8.8.8.8", "tls", True)
        self.assertIn("proxy_set_header X-Accord-Client-IP $remote_addr;", value)
        self.assertIn("location = /metrics { return 404; }", value)
        self.assertNotIn("listen 0.0.0.0", value)
        self.assertIn("proxy_set_header Host 127.0.0.1:7892;", value)

    def test_bootstrap_never_serves_browser_before_tls(self):
        value = ingress.render("8.8.8.8", "bootstrap", True)
        self.assertNotIn("proxy_pass", value)
        self.assertNotIn("ssl_certificate", value)
