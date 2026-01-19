package shopify

// ProductsQuery fetches products with variants
const ProductsQuery = `
query getProducts($first: Int!, $after: String) {
  products(first: $first, after: $after) {
    pageInfo {
      hasNextPage
      endCursor
    }
    edges {
      node {
        id
        title
        variants(first: 250) {
          edges {
            node {
              id
              sku
              title
              price
            }
          }
        }
      }
    }
  }
}
`
