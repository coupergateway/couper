# json_encode

Returns a [JSON](https://www.json.org) serialization of the given various `val`.

## Syntax

```hcl
string json_encode(val various)
```

## Example

```hcl
json_encode(request.context.myJWT)
```

-----

## Navigation

* &#8673; [Functions](../functions.md)
* &#8672; [`json_decode`](json-decode.md)
* &#8674; [`jwt_sign`](jwt-sign.md)
