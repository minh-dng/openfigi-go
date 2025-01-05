// openfigi: a client for the [OpenFIGI API].
//
// 3 types of queries:
//   - Search
//   - Filter (Search but with total count)
//   - Mapping
//
// Instructions:
//
//  1. Construct a builder.
//
//     - Search and Filter use [BaseItemBuilder], then construct a [BaseItem].
//
//     - Mapping uses [MappingItemBuilder], then construct a [MappingItem],
//     a slice of [MappingItem] or a [MappingRequest].
//
//  2. Set the properties through setters. (".Set[...](...)")
//
//  3. Build the item: [BaseItemBuilder.Build], [MappingItemBuilder.Build].
//     The package will validate the content of the item, reducing bad API calls.
//
//  4. [optional] API Key, set with [SetAPIKey].
//
//  5. Use the client to make the request.
//
//     - [BaseItem.Search], [BaseItem.Filter], returning [SearchResponse] or [FilterResponse]
//
//     - [MappingRequest] use [MappingRequest.Fetch] returning [][SingleMappingResponse]
//
//     - [SearchResponse.Next], [FilterResponse.Next] to fetch the next page.
//
// [OpenFIGI API]: https://www.openfigi.com/api
package openfigi
