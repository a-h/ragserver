# jsonapi

A simple JSON API client.

## Usage

### Get

```go
resp, ok, err := jsonapi.Get[itemsGetResponse](ctx, "https://example.com/items", jsonapi.WithAuthorization("Bearer abc"))
```

### Post

```go
req := itemsPostRequest{
        Name: "Item 1",
}
type itemsPostResponse struct {
        ID string `json:"id"`
}
resp, err := jsonapi.Post[itemsPostRequest, itemsPostResponse](ctx, "https://example.com/items/post/404", req)
```
