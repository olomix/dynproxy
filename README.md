# DynProxy

Meta proxy server. Accept connections and forward them to random proxy
from list. Keep in list only valid and working proxies.

## Testing

`curl -i -x localhost:3128 --proxy-header "Proxy-Connection:" -H "Cache-Control: no-cache" http://lomaka.org.ua/t.txt`
