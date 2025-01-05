# openfigi

A client for the [OpenFIGI API].

## Types of queries

- Search
- Filter (Search but with total count)
- Mapping

## Instructions

1. Construct a builder.

   - Search and Filter use `BaseItemBuilder`, then construct a `BaseItem`.
   - Mapping uses `MappingItemBuilder`, then construct a `MappingItem`,
     a slice of `MappingItem` or a `MappingRequest`.

2. Set the properties through setters. (`.Set[...](...)`)

3. Build the item (`.Build()`). The package will validate the content of the item, reducing bad API calls.

4. [optional] API Key, set with `SetAPIKey(string)`.

5. Use the client to make the request.

   - `BaseItem` use `.[Search|Filter](query string, start string)`
     returning `SearchResponse` or `FilterResponse`
   - `MappingRequest` use `.Fetch()` returning a slice of `SingleMappingResponse`
   - `SearchResponse` and `FilterResponse` have a `.Next()` method to fetch the next page.
  
## Developing

- `make generate` to generate the constants and hashset for validation
- `make test` for testing, will run `make generate`

[OpenFIGI API]: https://www.openfigi.com/api
