# Go bindings for Ledger Nano

These are simple go bindings to communicate with custom Ledger Nano apps.
This wraps the USB HID layer and handles the ledger specific communication.

## CLI Usage

Send bytes to ledger (app 0x80, op 0x02, payload 0xf00d)

```
make vendor
make install
ledger 8002F00D
```


## API Usage

```
import "github.com/ethanfrey/ledger"

func PingLedger(msg []byte) ([]byte, error) {
    device, err := ledger.FindLedger()
    if err != nil {
        return nil, err
    }
    return device.Exchange(msg, 0)
}
```

## TODO

* Actually use the timeout parameter
* Build higher level constructs for other apps
