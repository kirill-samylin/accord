#!/usr/bin/env python3
"""Add only the Next UI routes to an existing reviewed web nginx include."""
import argparse
import pathlib
import re


MARKER = '# Accord Next UI; native and browser API routes remain on Go.'


def port(value):
    text = str(value)
    if not re.fullmatch(r'[1-9][0-9]{0,4}', text) or not 1024 <= int(text) <= 65535:
        raise ValueError('expected unprivileged canonical TCP port')
    if int(text) == 7892:
        raise ValueError('Next cannot share the Go listener')
    return int(text)


def render(content, listen):
    listen = port(listen)
    if MARKER in content or 'accord-next' in content:
        raise ValueError('Next routes already present; inspect before replacing')
    marker = '    location = / {'
    if content.count(marker) != 1:
        raise ValueError('expected exactly one reviewed web root location')
    start = content.index(marker)
    end = content.index('\n    }', start) + len('\n    }')
    original = content[start:end]
    if ('proxy_pass http://127.0.0.1:7892;' not in original or
            original.count('{') != 1 or original.count('}') != 1):
        raise ValueError('unexpected web root route; refusing a broad rewrite')
    if 'location /web/ {' not in content or 'client_max_body_size 52494336;' not in content:
        raise ValueError('reviewed direct Go upload routes required')
    proxy = f'''        proxy_pass http://127.0.0.1:{listen};
        proxy_http_version 1.1;
        proxy_set_header Host $http_host;
        proxy_set_header Connection "";
        proxy_set_header Forwarded "";
        proxy_set_header X-Forwarded-For "";
        proxy_set_header X-Forwarded-Host "";
        proxy_set_header X-Forwarded-Proto "";
        proxy_set_header X-Forwarded-Port "";
        proxy_set_header X-Real-IP "";
        proxy_set_header X-Accord-Client-IP "";
        proxy_buffering off;
        proxy_request_buffering off;
        proxy_cache off;
        proxy_max_temp_file_size 0;
        proxy_connect_timeout 3s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
        proxy_next_upstream off;
        proxy_redirect off;
'''
    routes = '    ' + MARKER + '\n    location = / { return 302 /app; }\n'
    for route in ('= /app', '^~ /app/', '^~ /_next/static/', '^~ /fonts/', '= /PAPERCLIP-LICENSE.txt'):
        routes += f'    location {route} {{\n' + proxy + '    }\n'
    return content[:start] + routes.rstrip() + content[end:]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--input', required=True, type=pathlib.Path)
    parser.add_argument('--port', required=True, type=port)
    args = parser.parse_args()
    print(render(args.input.read_text(), args.port), end='')


if __name__ == '__main__':
    main()
