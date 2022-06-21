package genesyscloud

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/mypurecloud/platform-client-sdk-go/v72/platformclientv2"
	"github.com/mypurecloud/terraform-provider-genesyscloud/genesyscloud/consistency_checker"
)

var (
	journeySegmentSchema = map[string]*schema.Schema{
		"is_active": {
			Description: "Whether or not the segment is active.",
			Type:        schema.TypeBool,
			Optional:    true,
		},
		"display_name": {
			Description: "The display name of the segment.",
			Type:        schema.TypeString,
			Required:    true,
		},
		"description": {
			Description: "A description of the segment.",
			Type:        schema.TypeString,
			Optional:    true,
		},
		"color": {
			Description: "The hexadecimal color value of the segment.",
			Type:        schema.TypeString,
			Required:    true,
		},
		"scope": {
			Description:  "The target entity that a segment applies to.Valid values: Session, Customer.",
			Type:         schema.TypeString,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: validation.StringInSlice([]string{"Session", "Customer"}, false),
		},
		"should_display_to_agent": {
			Description: "Whether or not the segment should be displayed to agent/supervisor users.",
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     nil,
			// Customer scope only supports false to this value
		},
		"context": {
			Description: "The context of the segment.",
			Type:        schema.TypeSet,
			Optional:    true,
			MaxItems:    1,
			Elem:        contextResource,
		},
		"journey": {
			Description: "The pattern of rules defining the segment.",
			Type:        schema.TypeSet,
			Optional:    true,
			MaxItems:    1,
			Elem:        journeyResource,
		},
		"external_segment": {
			Description: "Details of an entity corresponding to this segment in an external system.",
			Type:        schema.TypeSet,
			Optional:    true,
			MaxItems:    1,
			Elem:        externalSegmentResource,
		},
		"assignment_expiration_days": {
			Description: "Time, in days, from when the segment is assigned until it is automatically unassigned.",
			Type:        schema.TypeInt,
			Optional:    true,
		},
	}

	contextResource = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"patterns": {
				Description: "A list of one or more patterns to match.",
				Type:        schema.TypeSet,
				Required:    true,
				Elem:        contextPatternResource,
			},
		},
	}

	journeyResource = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"patterns": {
				Description: "A list of zero or more patterns to match.",
				Type:        schema.TypeSet,
				Required:    true,
				Elem:        journeyPatternResource,
			},
		},
	}

	externalSegmentResource = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": {
				Description: "Identifier for the external segment in the system where it originates from.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"name": {
				Description: "Name for the external segment in the system where it originates from.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"source": {
				Description:  "The external system where the segment originates from.Valid values: AdobeExperiencePlatform, Custom.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"AdobeExperiencePlatform", "Custom"}, false),
			},
		},
	}

	contextPatternResource = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"criteria": {
				Description: "A list of one or more criteria to satisfy.",
				Type:        schema.TypeSet,
				Required:    true,
				Elem:        contextCriteriaResource,
			},
		},
	}

	journeyPatternResource = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"criteria": {
				Description: "A list of one or more criteria to satisfy.",
				Type:        schema.TypeSet,
				Required:    true,
				Elem:        journeyCriteriaResource,
			},
			"count": {
				Description: "The number of times the pattern must match.",
				Type:        schema.TypeInt,
				Required:    true,
			},
			"stream_type": {
				Description:  "The stream type for which this pattern can be matched on.Valid values: Web, Custom, Conversation.",
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"Web" /*, "Custom", "Conversation"*/}, false), // Custom and Conversation seem not to be supported by the API despite the documentation
			},
			"session_type": {
				Description:  "The session type for which this pattern can be matched on.",
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"web"}, false), // custom value seems not to be supported by the API despite the documentation
			},
			"event_name": {
				Description: "The name of the event for which this pattern can be matched on.",
				Type:        schema.TypeString,
				Optional:    true,
			},
		},
	}

	contextCriteriaResource = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"key": {
				Description:  "The criteria key.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"device.category", "device.type", "device.osFamily", "browser.family", "browser.lang", "browser.version", "mktCampaign.source", "mktCampaign.medium", "mktCampaign.name", "mktCampaign.term", "mktCampaign.content", "mktCampaign.clickId", "mktCampaign.network", "geolocation.countryName", "geolocation.locality", "geolocation.region", "geolocation.postalCode", "geolocation.country", "ipOrganization", "referrer.url", "referrer.medium", "referrer.hostname", "authenticated"}, false),
			},
			"values": {
				Description: "The criteria values.",
				Type:        schema.TypeSet,
				Required:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"should_ignore_case": {
				Description: "Should criteria be case insensitive.",
				Type:        schema.TypeBool,
				Required:    true,
			},
			"operator": {
				Description:  "The comparison operator.Valid values: containsAll, containsAny, notContainsAll, notContainsAny, equal, notEqual, greaterThan, greaterThanOrEqual, lessThan, lessThanOrEqual, startsWith, endsWith.",
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"containsAll", "containsAny", "notContainsAll", "notContainsAny", "equal", "notEqual", "greaterThan", "greaterThanOrEqual", "lessThan", "lessThanOrEqual", "startsWith", "endsWith"}, false),
			},
			"entity_type": {
				Description:  "The entity to match the pattern against.Valid values: visit.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"visit"}, false),
			},
		},
	}

	journeyCriteriaResource = &schema.Resource{
		Schema: map[string]*schema.Schema{
			"key": {
				Description: "The criteria key.",
				Type:        schema.TypeString,
				Required:    true,
				ValidateFunc: validation.Any(
					validation.StringInSlice([]string{"eventName", "page.url", "page.title", "page.hostname", "page.domain", "page.fragment", "page.keywords", "page.pathname", "searchQuery", "page.queryString"}, false),
					validation.StringMatch(func() *regexp.Regexp {
						r, _ := regexp.Compile("attributes\\..*\\.value")
						return r
					}(), ""),
				),
			},
			"values": {
				Description: "The criteria values.",
				Type:        schema.TypeSet,
				Required:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"should_ignore_case": {
				Description: "Should criteria be case insensitive.",
				Type:        schema.TypeBool,
				Required:    true,
			},
			"operator": {
				Description:  "The comparison operator.Valid values: containsAll, containsAny, notContainsAll, notContainsAny, equal, notEqual, greaterThan, greaterThanOrEqual, lessThan, lessThanOrEqual, startsWith, endsWith.",
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"containsAll", "containsAny", "notContainsAll", "notContainsAny", "equal", "notEqual", "greaterThan", "greaterThanOrEqual", "lessThan", "lessThanOrEqual", "startsWith", "endsWith"}, false),
			},
		},
	}
)

func getAllJourneySegments(_ context.Context, clientConfig *platformclientv2.Configuration) (ResourceIDMetaMap, diag.Diagnostics) {
	resources := make(ResourceIDMetaMap)
	journeyApi := platformclientv2.NewJourneyApiWithConfig(clientConfig)

	pageCount := 1 // Needed because of broken journey common paging
	for pageNum := 1; pageNum <= pageCount; pageNum++ {
		const pageSize = 100
		journeySegments, _, getErr := journeyApi.GetJourneySegments("", pageSize, pageNum, true, nil, nil, "")
		if getErr != nil {
			return nil, diag.Errorf("Failed to get page of journey segments: %v", getErr)
		}

		if journeySegments.Entities == nil || len(*journeySegments.Entities) == 0 {
			break
		}

		for _, journeySegment := range *journeySegments.Entities {
			resources[*journeySegment.Id] = &ResourceMeta{Name: *journeySegment.DisplayName}
		}

		pageCount = *journeySegments.PageCount
	}

	return resources, nil
}

func journeySegmentExporter() *ResourceExporter {
	return &ResourceExporter{
		GetResourcesFunc: getAllWithPooledClient(getAllJourneySegments),
		RefAttrs:         map[string]*RefAttrSettings{}, // No references
	}
}

func resourceJourneySegment() *schema.Resource {
	return &schema.Resource{
		Description: "Genesys Cloud Journey Segment",

		CreateContext: createWithPooledClient(createJourneySegment),
		ReadContext:   readWithPooledClient(readJourneySegment),
		UpdateContext: updateWithPooledClient(updateJourneySegment),
		DeleteContext: deleteWithPooledClient(deleteJourneySegment),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 1,
		Schema:        journeySegmentSchema,
	}
}

func createJourneySegment(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*providerMeta).ClientConfig
	journeyApi := platformclientv2.NewJourneyApiWithConfig(sdkConfig)
	journeySegment := buildSdkJourneySegment(d)

	log.Printf("Creating journey segment %s", *journeySegment.DisplayName)

	result, resp, err := journeyApi.PostJourneySegments(*journeySegment)
	if err != nil {
		return diag.Errorf("failed to create journey segment %s: %s\n(input: %+v)\n(resp: %s)", *journeySegment.DisplayName, err, *journeySegment, resp.RawBody)
	}

	d.SetId(*result.Id)

	log.Printf("Created journey segment %s %s", *result.DisplayName, *result.Id)
	return readJourneySegment(ctx, d, meta)
}

func readJourneySegment(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*providerMeta).ClientConfig
	journeyApi := platformclientv2.NewJourneyApiWithConfig(sdkConfig)

	log.Printf("Reading journey segment %s", d.Id())
	return withRetriesForRead(ctx, d, func() *resource.RetryError {
		journeySegment, resp, getErr := journeyApi.GetJourneySegment(d.Id())
		if getErr != nil {
			if isStatus404(resp) {
				return resource.RetryableError(fmt.Errorf("failed to read journey segment %s: %s", d.Id(), getErr))
			}
			return resource.NonRetryableError(fmt.Errorf("failed to read journey segment %s: %s", d.Id(), getErr))
		}

		cc := consistency_checker.NewConsistencyCheck(ctx, d, meta, resourceJourneySegment())
		flattenJourneySegment(d, journeySegment)

		log.Printf("Read journey segment %s %s", d.Id(), *journeySegment.DisplayName)
		return cc.CheckState()
	})
}

func updateJourneySegment(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	sdkConfig := meta.(*providerMeta).ClientConfig
	journeyApi := platformclientv2.NewJourneyApiWithConfig(sdkConfig)
	journeySegment := buildSdkPatchSegment(d)

	log.Printf("Updating journey segment %s", d.Id())
	if _, resp, err := journeyApi.PatchJourneySegment(d.Id(), *journeySegment); err != nil {
		return diag.Errorf("Error updating journey segment %s: %s\n(input: %+v)\n(resp: %s)", *journeySegment.DisplayName, err, *journeySegment, resp.RawBody)
	}

	log.Printf("Updated journey segment %s", d.Id())
	return readJourneySegment(ctx, d, meta)
}

func deleteJourneySegment(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	displayName := d.Get("display_name").(string)

	sdkConfig := meta.(*providerMeta).ClientConfig
	journeyApi := platformclientv2.NewJourneyApiWithConfig(sdkConfig)

	log.Printf("Deleting jounrey segment with display name %s", displayName)
	if _, err := journeyApi.DeleteJourneySegment(d.Id()); err != nil {
		return diag.Errorf("Failed to delete journey segment with display name %s: %s", displayName, err)
	}

	return withRetries(ctx, 30*time.Second, func() *resource.RetryError {
		_, resp, err := journeyApi.GetJourneySegment(d.Id())
		if err != nil {
			if isStatus404(resp) {
				// journey segment deleted
				log.Printf("Deleted journey segment %s", d.Id())
				return nil
			}
			return resource.NonRetryableError(fmt.Errorf("error deleting journey segment %s: %s", d.Id(), err))
		}

		return resource.RetryableError(fmt.Errorf("journey segment %s still exists", d.Id()))
	})
}

func flattenJourneySegment(d *schema.ResourceData, journeySegment *platformclientv2.Journeysegment) {
	d.Set("display_name", *journeySegment.DisplayName)
	setNullableValue(d, "description", journeySegment.Description)
	setNullableValue(d, "color", journeySegment.Color)
	setNullableValue(d, "scope", journeySegment.Scope)
	setNullableValue(d, "should_display_to_agent", journeySegment.ShouldDisplayToAgent)
	setNullableValue(d, "context", flattenGenericAsList(journeySegment.Context, flattenContext))
	setNullableValue(d, "journey", flattenGenericAsList(journeySegment.Journey, flattenJourney))
	setNullableValue(d, "external_segment", flattenGenericAsList(journeySegment.ExternalSegment, flattenExternalSegment))
	setNullableValue(d, "assignment_expiration_days", journeySegment.AssignmentExpirationDays)
}

func buildSdkJourneySegment(journeySegment *schema.ResourceData) *platformclientv2.Journeysegment {
	isActive := getNullableBool(journeySegment, "is_active")
	displayName := getNullableValue[string](journeySegment, "display_name")
	description := getNullableValue[string](journeySegment, "description")
	color := getNullableValue[string](journeySegment, "color")
	scope := getNullableValue[string](journeySegment, "scope")
	shouldDisplayToAgent := getNullableBool(journeySegment, "should_display_to_agent")
	sdkContext := buildSdkGenericListFirstElement(journeySegment, "context", buildSdkContext)
	journey := buildSdkGenericListFirstElement(journeySegment, "journey", buildSdkJourney)
	externalSegment := buildSdkGenericListFirstElement(journeySegment, "external_segment", buildSdkExternalSegment)
	assignmentExpirationDays := getNullableValue[int](journeySegment, "assignment_expiration_days")

	return &platformclientv2.Journeysegment{
		IsActive:                 isActive,
		DisplayName:              displayName,
		Description:              description,
		Color:                    color,
		Scope:                    scope,
		ShouldDisplayToAgent:     shouldDisplayToAgent,
		Context:                  sdkContext,
		Journey:                  journey,
		ExternalSegment:          externalSegment,
		AssignmentExpirationDays: assignmentExpirationDays,
	}
}

func buildSdkPatchSegment(journeySegment *schema.ResourceData) *platformclientv2.Patchsegment {
	isActive := getNullableBool(journeySegment, "is_active")
	displayName := getNullableValue[string](journeySegment, "display_name")
	description := getNullableValue[string](journeySegment, "description")
	color := getNullableValue[string](journeySegment, "color")
	shouldDisplayToAgent := getNullableBool(journeySegment, "should_display_to_agent")
	sdkContext := buildSdkGenericListFirstElement(journeySegment, "context", buildSdkContext)
	journey := buildSdkGenericListFirstElement(journeySegment, "journey", buildSdkJourney)
	externalSegment := buildSdkGenericListFirstElement(journeySegment, "external_segment", buildSdkPatchExternalSegment)
	assignmentExpirationDays := getNullableValue[int](journeySegment, "assignment_expiration_days")

	return &platformclientv2.Patchsegment{
		IsActive:                 isActive,
		DisplayName:              displayName,
		Description:              description,
		Color:                    color,
		ShouldDisplayToAgent:     shouldDisplayToAgent,
		Context:                  sdkContext,
		Journey:                  journey,
		ExternalSegment:          externalSegment,
		AssignmentExpirationDays: assignmentExpirationDays,
	}
}

func flattenContext(context *platformclientv2.Context) map[string]interface{} {
	if len(*context.Patterns) == 0 {
		return nil
	}
	contextMap := make(map[string]interface{})
	contextMap["patterns"] = flattenGenericList(context.Patterns, flattenContextPattern)
	return contextMap
}

func buildSdkContext(context map[string]interface{}) *platformclientv2.Context {
	patterns := &[]platformclientv2.Contextpattern{}
	if context != nil {
		patterns = buildSdkGenericList(context, "patterns", buildSdkContextPattern)
	}
	return &platformclientv2.Context{
		Patterns: patterns,
	}
}

func flattenContextPattern(contextPattern *platformclientv2.Contextpattern) map[string]interface{} {
	contextPatternMap := make(map[string]interface{})
	contextPatternMap["criteria"] = flattenGenericList(contextPattern.Criteria, flattenEntityTypeCriteria)
	return contextPatternMap
}

func buildSdkContextPattern(contextPattern map[string]interface{}) *platformclientv2.Contextpattern {
	return &platformclientv2.Contextpattern{
		Criteria: buildSdkGenericList(contextPattern, "criteria", buildSdkEntityTypeCriteria),
	}
}

func flattenEntityTypeCriteria(entityTypeCriteria *platformclientv2.Entitytypecriteria) map[string]interface{} {
	entityTypeCriteriaMap := make(map[string]interface{})
	if entityTypeCriteria.Key != nil {
		entityTypeCriteriaMap["key"] = *entityTypeCriteria.Key
	}
	if entityTypeCriteria.Values != nil {
		entityTypeCriteriaMap["values"] = stringListToSet(*entityTypeCriteria.Values)
	}
	if entityTypeCriteria.ShouldIgnoreCase != nil {
		entityTypeCriteriaMap["should_ignore_case"] = *entityTypeCriteria.ShouldIgnoreCase
	}
	if entityTypeCriteria.Operator != nil {
		entityTypeCriteriaMap["operator"] = *entityTypeCriteria.Operator
	}
	if entityTypeCriteria.EntityType != nil {
		entityTypeCriteriaMap["entity_type"] = *entityTypeCriteria.EntityType
	}
	return entityTypeCriteriaMap
}

func buildSdkEntityTypeCriteria(entityTypeCriteria map[string]interface{}) *platformclientv2.Entitytypecriteria {
	key := entityTypeCriteria["key"].(string)
	values := buildSdkStringListFromMapEntry(entityTypeCriteria, "values")
	shouldIgnoreCase := entityTypeCriteria["should_ignore_case"].(bool)
	operator := entityTypeCriteria["operator"].(string)
	entityType := entityTypeCriteria["entity_type"].(string)

	return &platformclientv2.Entitytypecriteria{
		Key:              &key,
		Values:           values,
		ShouldIgnoreCase: &shouldIgnoreCase,
		Operator:         &operator,
		EntityType:       &entityType,
	}
}

func flattenJourney(journey *platformclientv2.Journey) map[string]interface{} {
	if len(*journey.Patterns) == 0 {
		return nil
	}
	journeyMap := make(map[string]interface{})
	journeyMap["patterns"] = flattenGenericList(journey.Patterns, flattenJourneyPattern)
	return journeyMap
}

func buildSdkJourney(journey map[string]interface{}) *platformclientv2.Journey {
	patterns := &[]platformclientv2.Journeypattern{}
	if journey != nil {
		patterns = buildSdkGenericList(journey, "patterns", buildSdkJourneyPattern)
	}
	return &platformclientv2.Journey{
		Patterns: patterns,
	}
}

func flattenJourneyPattern(journeyPattern *platformclientv2.Journeypattern) map[string]interface{} {
	journeyPatternMap := make(map[string]interface{})
	journeyPatternMap["criteria"] = flattenGenericList(journeyPattern.Criteria, flattenCriteria)
	if journeyPattern.Count != nil {
		journeyPatternMap["count"] = *journeyPattern.Count
	}
	if journeyPattern.StreamType != nil {
		journeyPatternMap["stream_type"] = *journeyPattern.StreamType
	}
	if journeyPattern.SessionType != nil {
		journeyPatternMap["session_type"] = *journeyPattern.SessionType
	}
	if journeyPattern.EventName != nil {
		journeyPatternMap["event_name"] = *journeyPattern.EventName
	}
	return journeyPatternMap
}

func buildSdkJourneyPattern(journeyPattern map[string]interface{}) *platformclientv2.Journeypattern {
	criteria := buildSdkGenericList(journeyPattern, "criteria", buildSdkCriteria)
	count := journeyPattern["count"].(int)
	streamType := journeyPattern["stream_type"].(string)
	sessionType := journeyPattern["session_type"].(string)
	eventName := journeyPattern["event_name"].(string)

	return &platformclientv2.Journeypattern{
		Criteria:    criteria,
		Count:       &count,
		StreamType:  &streamType,
		SessionType: &sessionType,
		EventName:   &eventName,
	}
}

func flattenCriteria(criteria *platformclientv2.Criteria) map[string]interface{} {
	criteriaMap := make(map[string]interface{})
	if criteria.Key != nil {
		criteriaMap["key"] = *criteria.Key
	}
	if criteria.Values != nil {
		criteriaMap["values"] = stringListToSet(*criteria.Values)
	}
	if criteria.ShouldIgnoreCase != nil {
		criteriaMap["should_ignore_case"] = *criteria.ShouldIgnoreCase
	}
	if criteria.Operator != nil {
		criteriaMap["operator"] = *criteria.Operator
	}
	return criteriaMap
}

func buildSdkCriteria(criteria map[string]interface{}) *platformclientv2.Criteria {
	key := criteria["key"].(string)
	values := buildSdkStringListFromMapEntry(criteria, "values")
	shouldIgnoreCase := criteria["should_ignore_case"].(bool)
	operator := criteria["operator"].(string)

	return &platformclientv2.Criteria{
		Key:              &key,
		Values:           values,
		ShouldIgnoreCase: &shouldIgnoreCase,
		Operator:         &operator,
	}
}

func flattenExternalSegment(externalSegment *platformclientv2.Externalsegment) map[string]interface{} {
	externalSegmentMap := make(map[string]interface{})
	if externalSegment.Id != nil {
		externalSegmentMap["id"] = *externalSegment.Id
	}
	if externalSegment.Name != nil {
		externalSegmentMap["name"] = *externalSegment.Name
	}
	if externalSegment.Source != nil {
		externalSegmentMap["source"] = *externalSegment.Source
	}
	return externalSegmentMap
}

func buildSdkExternalSegment(externalSegment map[string]interface{}) *platformclientv2.Externalsegment {
	if externalSegment == nil {
		return nil
	}

	name := externalSegment["name"].(string)
	source := externalSegment["source"].(string)

	return &platformclientv2.Externalsegment{
		Name:   &name,
		Source: &source,
	}
}

func buildSdkPatchExternalSegment(externalSegment map[string]interface{}) *platformclientv2.Patchexternalsegment {
	if externalSegment == nil {
		return nil
	}

	name := externalSegment["name"].(string)

	return &platformclientv2.Patchexternalsegment{
		Name: &name,
	}
}