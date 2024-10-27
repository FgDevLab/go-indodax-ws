Example Implementation Of Indodax Market Web Socket https://github.com/btcid/indodax-official-api-docs/blob/master/Marketdata-websocket.md

- **Authentication**: The client sends an authentication token to the WebSocket server.
- **Ping-Pong**: The client sends ping messages every 30 seconds to keep the connection alive.
- **Subscription**: After authenticating, the client subscribes to the `chart:tick-btcidr` channel to receive price updates.

## Example Output

```
╔════════════════════════════════════════╗
║           BTC/IDR LIVE PRICE           ║
╠════════════════════════════════════════╣
║ DATE   : 24-10-2024 14:30:12           ║
║ PRICE  : Rp 987,654,321                ║
║ VOLUME : 0.123 BTC                     ║
╚════════════════════════════════════════╝
```
