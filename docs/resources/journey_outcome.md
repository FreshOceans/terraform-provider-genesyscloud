---
page_title: "genesyscloud_journey_outcome Resource - terraform-provider-genesyscloud"
subcategory: ""
description: |-
  Genesys Cloud Journey Outcome
---
# genesyscloud_journey_outcome (Resource)

Genesys Cloud Journey Outcome

## API Usage
The following Genesys Cloud APIs are used by this resource. Ensure your OAuth Client has been granted the necessary scopes and permissions to perform these operations:

* [GET /api/v2/journey/outcomes](https://developer.genesys.cloud/commdigital/digital/webmessaging/journey/journey-apis#get-api-v2-journey-outcomes)
* [POST /api/v2/journey/outcomes](https://developer.genesys.cloud/commdigital/digital/webmessaging/journey/journey-apis#post-api-v2-journey-outcomes)
* [GET /api/v2/journey/outcomes/{outcomeId}](https://developer.genesys.cloud/commdigital/digital/webmessaging/journey/journey-apis#get-api-v2-journey-outcomes--outcomeId-)
* [PATCH /api/v2/journey/outcomes/{outcomeId}](https://developer.genesys.cloud/commdigital/digital/webmessaging/journey/journey-apis#patch-api-v2-journey-outcomes--outcomeId-)
* [DELETE /api/v2/journey/outcomes/{outcomeId}](https://developer.genesys.cloud/commdigital/digital/webmessaging/journey/journey-apis#delete-api-v2-journey-outcomes--outcomeId-)

## Example Usage

```terraform
resource "genesyscloud_journey_outcome" "example_journey_outcome_resource" {
  is_active    = true
  display_name = "example journey outcome name"
  description  = "description of journey outcome"
  is_positive  = true
  journey {
    patterns {
      criteria {
        key                = "page.title"
        values             = ["Title"]
        operator           = "notEqual"
        should_ignore_case = true
      }
      count        = 1
      stream_type  = "Web"
      session_type = "web"
    }
  }
  context {
    patterns {
      criteria {
        key                = "geolocation.postalCode"
        values             = ["something"]
        operator           = "equal"
        should_ignore_case = true
        entity_type        = "visit"
      }
    }
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `display_name` (String) The display name of the outcome.

### Optional

- `associated_value_field` (Block Set, Max: 1) The field from the event indicating the associated value. Associated_value_field needs `eventtypes` to be created, which is a feature coming soon. More details available here:  https://developer.genesys.cloud/commdigital/digital/webmessaging/journey/eventtypes  https://all.docs.genesys.com/ATC/Current/AdminGuide/Custom_sessions (see [below for nested schema](#nestedblock--associated_value_field))
- `context` (Block Set, Max: 1) The context of the outcome. (see [below for nested schema](#nestedblock--context))
- `description` (String) A description of the outcome.
- `is_active` (Boolean) Whether or not the outcome is active. Defaults to `true`.
- `is_positive` (Boolean) Whether or not the outcome is positive. Defaults to `true`.
- `journey` (Block Set, Max: 1) The pattern of rules defining the outcome. (see [below for nested schema](#nestedblock--journey))

### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedblock--associated_value_field"></a>
### Nested Schema for `associated_value_field`

Required:

- `data_type` (String) The data type of the value field.Valid values: Number, Integer.
- `name` (String) The field name for extracting value from event.


<a id="nestedblock--context"></a>
### Nested Schema for `context`

Required:

- `patterns` (Block Set, Min: 1) A list of one or more patterns to match. (see [below for nested schema](#nestedblock--context--patterns))

<a id="nestedblock--context--patterns"></a>
### Nested Schema for `context.patterns`

Required:

- `criteria` (Block Set, Min: 1) A list of one or more criteria to satisfy. (see [below for nested schema](#nestedblock--context--patterns--criteria))

<a id="nestedblock--context--patterns--criteria"></a>
### Nested Schema for `context.patterns.criteria`

Required:

- `entity_type` (String) The entity to match the pattern against.Valid values: visit.
- `key` (String) The criteria key.
- `should_ignore_case` (Boolean) Should criteria be case insensitive.
- `values` (Set of String) The criteria values.

Optional:

- `operator` (String) The comparison operator. Valid values: containsAll, containsAny, notContainsAll, notContainsAny, equal, notEqual, greaterThan, greaterThanOrEqual, lessThan, lessThanOrEqual, startsWith, endsWith. Defaults to `equal`.




<a id="nestedblock--journey"></a>
### Nested Schema for `journey`

Required:

- `patterns` (Block Set, Min: 1) A list of zero or more patterns to match. (see [below for nested schema](#nestedblock--journey--patterns))

<a id="nestedblock--journey--patterns"></a>
### Nested Schema for `journey.patterns`

Required:

- `count` (Number) The number of times the pattern must match.
- `criteria` (Block Set, Min: 1) A list of one or more criteria to satisfy. (see [below for nested schema](#nestedblock--journey--patterns--criteria))
- `session_type` (String) The session type for which this pattern can be matched on.
- `stream_type` (String) The stream type for which this pattern can be matched on.Valid values: Web, Custom, Conversation.

Optional:

- `event_name` (String) The name of the event for which this pattern can be matched on.

<a id="nestedblock--journey--patterns--criteria"></a>
### Nested Schema for `journey.patterns.criteria`

Required:

- `key` (String) The criteria key.
- `should_ignore_case` (Boolean) Should criteria be case insensitive.
- `values` (Set of String) The criteria values.

Optional:

- `operator` (String) The comparison operator.Valid values: containsAll, containsAny, notContainsAll, notContainsAny, equal, notEqual, greaterThan, greaterThanOrEqual, lessThan, lessThanOrEqual, startsWith, endsWith. Defaults to `equal`.
