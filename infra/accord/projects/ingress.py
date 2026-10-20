"""Narrow transformation of the reviewed Next ingress; preserve existing limits."""
import argparse
from pathlib import Path

NATIVE = '    location ~ ^/(api/v1(?:/|$)|mcp$|readyz$|healthz$) {'
SCOPED_NATIVE = '    location ~ "^/(api/v1(?:/|$)|mcp(?:$|/p/p_[0-9a-f]{32}$)|readyz$|healthz$)" {'
UPLOAD = '    location = /web/designs {'
SCOPED_UPLOAD = '    location ~ "^/web/p/p_[0-9a-f]{32}/designs$" {'


def render(source):
    if (source.count(NATIVE) != 1 or source.count(UPLOAD) != 1 or
            '    location ^~ /app/ {' not in source or
            'limit_req zone=accord_next_assets burst=100 nodelay;' not in source or
            SCOPED_UPLOAD in source or SCOPED_NATIVE in source):
        raise ValueError('expected single reviewed Next/native/browser ingress')
    start = source.index(UPLOAD)
    end = source.index('\n    }', start) + len('\n    }')
    block = source[start:end]
    required = ('client_max_body_size 52494336;', 'client_body_timeout 120s;',
                'proxy_pass http://127.0.0.1:7892;', 'proxy_request_buffering off;',
                'proxy_set_header X-Accord-Client-IP $remote_addr;')
    if any(block.count(value) != 1 for value in required):
        raise ValueError('reviewed upload limits or trusted ingress differ')
    scoped = block.replace(UPLOAD, SCOPED_UPLOAD, 1)
    return (source[:end] + '\n' + scoped + source[end:]).replace(NATIVE, SCOPED_NATIVE, 1)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--input', type=Path, required=True)
    args = parser.parse_args()
    print(render(args.input.read_text()), end='')


if __name__ == '__main__':
    main()
