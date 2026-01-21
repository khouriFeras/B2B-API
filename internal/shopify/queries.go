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

// OrderByNumberQueryTemplate fetches an order by its order number
// Note: The query parameter must be a string literal, not a variable
// So we'll build the query string dynamically using fmt.Sprintf
const OrderByNumberQueryTemplate = `
query getOrderByNumber {
  orders(first: 1, query: "%s") {
    edges {
      node {
        id
        name
        displayFulfillmentStatus
        displayFinancialStatus
        createdAt
        updatedAt
        totalPriceSet {
          shopMoney {
            amount
            currencyCode
          }
        }
        customer {
          firstName
          lastName
          email
          phone
        }
        shippingAddress {
          address1
          address2
          city
          province
          zip
          country
        }
        lineItems(first: 250) {
          edges {
            node {
              id
              title
              quantity
              variant {
                id
                sku
                title
                price
              }
              originalUnitPriceSet {
                shopMoney {
                  amount
                  currencyCode
                }
              }
            }
          }
        }
        fulfillments {
          id
          status
          trackingInfo {
            number
            url
            company
          }
        }
      }
    }
  }
}
`

// OrderByIDQuery fetches an order by its Shopify GID
const OrderByIDQuery = `
query getOrderByID($id: ID!) {
  node(id: $id) {
    ... on Order {
      id
      name
      displayFulfillmentStatus
      displayFinancialStatus
      createdAt
      updatedAt
      totalPriceSet {
        shopMoney {
          amount
          currencyCode
        }
      }
      customer {
        firstName
        lastName
        email
        phone
      }
      shippingAddress {
        address1
        address2
        city
        province
        zip
        country
      }
      lineItems(first: 250) {
        edges {
          node {
            id
            title
            quantity
            variant {
              id
              sku
              title
              price
            }
            originalUnitPriceSet {
              shopMoney {
                amount
                currencyCode
              }
            }
          }
        }
      }
      fulfillments {
        id
        status
        trackingInfo {
          number
          url
          company
        }
      }
    }
  }
}
`