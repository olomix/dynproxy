# DynProxy

Meta proxy server. Accept connections and forward them to random proxy
from list. Keep in list only valid and working proxies.

Dynproxy adds special header to reply "X-Dynproxy-Proxy". It is used to 
report which proxy was chosen. Client may set this header too to force dynproxy
use specified proxy.

## Testing

`curl -i -x localhost:3128 --proxy-header "Proxy-Connection:" -H "Cache-Control: no-cache" http://lomaka.org.ua/t.txt`
