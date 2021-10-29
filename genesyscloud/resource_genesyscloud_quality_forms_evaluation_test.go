package genesyscloud

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/mypurecloud/platform-client-sdk-go/v56/platformclientv2"
)

type evaluationFormStruct struct {
	name           string
	published      bool
	questionGroups []evaluationFormQuestionGroupStruct
}

type evaluationFormQuestionGroupStruct struct {
	name                    string
	defaultAnswersToHighest bool
	defaultAnswersToNA      bool
	naEnabled               bool
	weight                  float32
	manualWeight            bool
	questions               []evaluationFormQuestionStruct
	visibilityCondition     visibilityConditionStruct
}

type evaluationFormQuestionStruct struct {
	text                string
	helpText            string
	naEnabled           bool
	commentsRequired    bool
	isKill              bool
	isCritical          bool
	visibilityCondition visibilityConditionStruct
	answerOptions       []answerOptionStruct
}

type answerOptionStruct struct {
	text  string
	value int
}

type visibilityConditionStruct struct {
	combiningOperation string
	predicates         []string
}

func TestAccResourceEvaluationFormBasic(t *testing.T) {
	formResource1 := "test-evaluation-form-1"

	evaluationForm1 := evaluationFormStruct{
		name: "terraform-form-evaluations-" + uuid.NewString(),
		questionGroups: []evaluationFormQuestionGroupStruct{
			{
				name:   "Test Question",
				weight: 1,
				questions: []evaluationFormQuestionStruct{
					{
						text: "Did the agent perform the opening spiel?",
						answerOptions: []answerOptionStruct{
							{
								text:  "Yes",
								value: 1,
							},
							{
								text:  "No",
								value: 0,
							},
						},
					},
				},
			},
		},
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				// Create
				Config: generateEvaluationFormResource(formResource1, &evaluationForm1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("resource_genesyscloud_quality_forms_evaluation."+formResource1, "name", evaluationForm1.name),
				),
			},
			{
				// Import/Read
				ResourceName:      "resource_genesyscloud_quality_forms_evaluation." + formResource1,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
		CheckDestroy: testVerifyEvaluationFormDestroyed,
	})
}

func testVerifyEvaluationFormDestroyed(state *terraform.State) error {
	qualityAPI := platformclientv2.NewQualityApi()
	for _, rs := range state.RootModule().Resources {
		if rs.Type != "resource_genesyscloud_quality_forms_evaluation" {
			continue
		}

		form, resp, err := qualityAPI.GetQualityFormsEvaluation(rs.Primary.ID)
		if form != nil {
			continue
		}

		if form != nil {
			return fmt.Errorf("Evaluation form (%s) still exists", rs.Primary.ID)
		}

		if isStatus404(resp) {
			// Evaluation form not found as expected
			continue
		}

		// Unexpected error
		return fmt.Errorf("Unexpected error: %s", err)
	}
	// Success. All Evaluation forms destroyed
	return nil
}

func generateEvaluationFormResource(resourceID string, evaluationForm *evaluationFormStruct) string {
	return fmt.Sprintf(`resource "resource_genesyscloud_quality_forms_evaluation" "%s" {
		name = "%s"
		published = %v
		question_groups = %s
	}
	`, resourceID,
		evaluationForm.name,
		evaluationForm.published,
		generateEvaluationFormQuestionGroups(&evaluationForm.questionGroups),
	)
}

func generateEvaluationFormQuestionGroups(questionGroups *[]evaluationFormQuestionGroupStruct) string {
	if questionGroups == nil {
		return nullValue
	}

	questionGroupsString := ""

	for i, questionGroup := range *questionGroups {
		questionGroupString := ""

		if i > 0 {
			questionGroupString += ", "
		}

		questionGroupString += fmt.Sprintf(`{
            name = "%s"
            default_answers_to_highest = %v
            default_answers_to_na  = %v
            na_enabled = %v
            weight = %v
            manual_weight = %v
            questions = %s
            visibility_condition = %s
        }`, questionGroup.name,
			questionGroup.defaultAnswersToHighest,
			questionGroup.defaultAnswersToNA,
			questionGroup.naEnabled,
			questionGroup.weight,
			questionGroup.manualWeight,
			generateEvaluationFormQuestions(&questionGroup.questions),
			generateFormVisibilityCondition(&questionGroup.visibilityCondition),
		)

		questionGroupsString += questionGroupString
	}

	return fmt.Sprintf(`[%s]`, questionGroupsString)
}

func generateEvaluationFormQuestions(questions *[]evaluationFormQuestionStruct) string {
	if questions == nil {
		return nullValue
	}

	questionsString := ""

	for i, question := range *questions {
		questionString := ""

		if i > 0 {
			questionString += ", "
		}

		questionString += fmt.Sprintf(`{
            text = "%s"
            help_text = "%s"
            na_enabled = %v
            comments_required = %v
            is_kill = %v
            is_critical = %v
            visibility_condition = %s
            answer_options = %s
        }`, question.text,
			question.helpText,
			question.naEnabled,
			question.commentsRequired,
			question.isKill,
			question.isCritical,
			generateFormVisibilityCondition(&question.visibilityCondition),
			generateFormAnswerOptions(&question.answerOptions),
		)

		questionsString += questionString
	}

	return fmt.Sprintf(`[%s]`, questionsString)
}

func generateFormAnswerOptions(answerOptions *[]answerOptionStruct) string {
	if answerOptions == nil {
		return nullValue
	}

	answerOptionsString := ""

	for i, answerOption := range *answerOptions {
		answerOptionString := ""

		if i > 0 {
			answerOptionString += ", "
		}

		answerOptionString += fmt.Sprintf(`{
            name = "%s"
            defaultAnswersToHighest = %d
        }`, answerOption.text,
			answerOption.value,
		)

		answerOptionsString += answerOptionString
	}

	return fmt.Sprintf(`[%s]`, answerOptionsString)
}

func generateFormVisibilityCondition(condition *visibilityConditionStruct) string {
	if condition == nil {
		return nullValue
	}

	predicateString := ""

	for i, predicate := range condition.predicates {
		if i > 0 {
			predicateString += ", "
		}

		predicateString += strconv.Quote(predicate)
	}

	return fmt.Sprintf(`{
        combining_operation = "%s"
        predicates = [%s]
    }`, condition.combiningOperation,
		predicateString,
	)
}