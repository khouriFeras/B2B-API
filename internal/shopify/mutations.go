package shopify

// DraftOrderCreateMutation creates a draft order
const DraftOrderCreateMutation = `
mutation draftOrderCreate($input: DraftOrderInput!) {
  draftOrderCreate(input: $input) {
    draftOrder {
      id
      name
      order {
        id
      }
    }
    userErrors {
      field
      message
    }
  }
}
`

// DraftOrderInput represents the input for creating a draft order
type DraftOrderInput struct {
	LineItems     []DraftOrderLineItemInput `json:"lineItems"`
	CustomerID    *string                    `json:"customerId,omitempty"`
	Email         *string                    `json:"email,omitempty"`
	ShippingAddress *DraftOrderAddressInput `json:"shippingAddress,omitempty"`
	Tags          []string                   `json:"tags,omitempty"`
	Note          *string                   `json:"note,omitempty"`
	CustomAttributes []DraftOrderAttributeInput `json:"customAttributes,omitempty"`
}

type DraftOrderLineItemInput struct {
	VariantID    *string  `json:"variantId,omitempty"`
	Title        *string  `json:"title,omitempty"`
	Price        *string  `json:"price,omitempty"`
	Quantity     int      `json:"quantity"`
	CustomAttributes []DraftOrderAttributeInput `json:"customAttributes,omitempty"`
}

type DraftOrderAddressInput struct {
	FirstName    string  `json:"firstName"`
	LastName     *string `json:"lastName,omitempty"`
	Address1     string  `json:"address1"`
	Address2     *string `json:"address2,omitempty"`
	City         string  `json:"city"`
	Province     *string `json:"province,omitempty"`
	Zip          string  `json:"zip"`
	Country      string  `json:"country"`
	Phone        *string `json:"phone,omitempty"`
}

type DraftOrderAttributeInput struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
