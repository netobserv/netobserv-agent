/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

// This file contains a collection of methods that can be used from go-restful to
// generate Swagger API documentation for its models. Please read this PR for more
// information on the implementation: https://github.com/emicklei/go-restful/pull/215
//
// TODOs are ignored from the parser (e.g. TODO(andronat):... || TODO:...) if and only if
// they are on one line! For multiple line or blocks that you want to ignore use ---.
// Any context after a --- is ignored.
//
// Those methods can be generated by using hack/update-codegen.sh

// AUTO-GENERATED FUNCTIONS START HERE. DO NOT EDIT.
var map_CronJob = map[string]string{
	"":         "CronJob represents the configuration of a single cron job.",
	"metadata": "Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata",
	"spec":     "Specification of the desired behavior of a cron job, including the schedule. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status",
	"status":   "Current status of a cron job. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status",
}

func (CronJob) SwaggerDoc() map[string]string {
	return map_CronJob
}

var map_CronJobList = map[string]string{
	"":         "CronJobList is a collection of cron jobs.",
	"metadata": "Standard list metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata",
	"items":    "items is the list of CronJobs.",
}

func (CronJobList) SwaggerDoc() map[string]string {
	return map_CronJobList
}

var map_CronJobSpec = map[string]string{
	"":                           "CronJobSpec describes how the job execution will look like and when it will actually run.",
	"schedule":                   "The schedule in Cron format, see https://en.wikipedia.org/wiki/Cron.",
	"timeZone":                   "The time zone name for the given schedule, see https://en.wikipedia.org/wiki/List_of_tz_database_time_zones. If not specified, this will default to the time zone of the kube-controller-manager process. The set of valid time zone names and the time zone offset is loaded from the system-wide time zone database by the API server during CronJob validation and the controller manager during execution. If no system-wide time zone database can be found a bundled version of the database is used instead. If the time zone name becomes invalid during the lifetime of a CronJob or due to a change in host configuration, the controller will stop creating new new Jobs and will create a system event with the reason UnknownTimeZone. More information can be found in https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/#time-zones",
	"startingDeadlineSeconds":    "Optional deadline in seconds for starting the job if it misses scheduled time for any reason.  Missed jobs executions will be counted as failed ones.",
	"concurrencyPolicy":          "Specifies how to treat concurrent executions of a Job. Valid values are:\n\n- \"Allow\" (default): allows CronJobs to run concurrently; - \"Forbid\": forbids concurrent runs, skipping next run if previous run hasn't finished yet; - \"Replace\": cancels currently running job and replaces it with a new one",
	"suspend":                    "This flag tells the controller to suspend subsequent executions, it does not apply to already started executions.  Defaults to false.",
	"jobTemplate":                "Specifies the job that will be created when executing a CronJob.",
	"successfulJobsHistoryLimit": "The number of successful finished jobs to retain. Value must be non-negative integer. Defaults to 3.",
	"failedJobsHistoryLimit":     "The number of failed finished jobs to retain. Value must be non-negative integer. Defaults to 1.",
}

func (CronJobSpec) SwaggerDoc() map[string]string {
	return map_CronJobSpec
}

var map_CronJobStatus = map[string]string{
	"":                   "CronJobStatus represents the current state of a cron job.",
	"active":             "A list of pointers to currently running jobs.",
	"lastScheduleTime":   "Information when was the last time the job was successfully scheduled.",
	"lastSuccessfulTime": "Information when was the last time the job successfully completed.",
}

func (CronJobStatus) SwaggerDoc() map[string]string {
	return map_CronJobStatus
}

var map_Job = map[string]string{
	"":         "Job represents the configuration of a single job.",
	"metadata": "Standard object's metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata",
	"spec":     "Specification of the desired behavior of a job. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status",
	"status":   "Current status of a job. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status",
}

func (Job) SwaggerDoc() map[string]string {
	return map_Job
}

var map_JobCondition = map[string]string{
	"":                   "JobCondition describes current state of a job.",
	"type":               "Type of job condition, Complete or Failed.",
	"status":             "Status of the condition, one of True, False, Unknown.",
	"lastProbeTime":      "Last time the condition was checked.",
	"lastTransitionTime": "Last time the condition transit from one status to another.",
	"reason":             "(brief) reason for the condition's last transition.",
	"message":            "Human readable message indicating details about last transition.",
}

func (JobCondition) SwaggerDoc() map[string]string {
	return map_JobCondition
}

var map_JobList = map[string]string{
	"":         "JobList is a collection of jobs.",
	"metadata": "Standard list metadata. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata",
	"items":    "items is the list of Jobs.",
}

func (JobList) SwaggerDoc() map[string]string {
	return map_JobList
}

var map_JobSpec = map[string]string{
	"":                        "JobSpec describes how the job execution will look like.",
	"parallelism":             "Specifies the maximum desired number of pods the job should run at any given time. The actual number of pods running in steady state will be less than this number when ((.spec.completions - .status.successful) < .spec.parallelism), i.e. when the work left to do is less than max parallelism. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/",
	"completions":             "Specifies the desired number of successfully finished pods the job should be run with.  Setting to null means that the success of any pod signals the success of all pods, and allows parallelism to have any positive value.  Setting to 1 means that parallelism is limited to 1 and the success of that pod signals the success of the job. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/",
	"activeDeadlineSeconds":   "Specifies the duration in seconds relative to the startTime that the job may be continuously active before the system tries to terminate it; value must be positive integer. If a Job is suspended (at creation or through an update), this timer will effectively be stopped and reset when the Job is resumed again.",
	"podFailurePolicy":        "Specifies the policy of handling failed pods. In particular, it allows to specify the set of actions and conditions which need to be satisfied to take the associated action. If empty, the default behaviour applies - the counter of failed pods, represented by the jobs's .status.failed field, is incremented and it is checked against the backoffLimit. This field cannot be used in combination with restartPolicy=OnFailure.",
	"successPolicy":           "successPolicy specifies the policy when the Job can be declared as succeeded. If empty, the default behavior applies - the Job is declared as succeeded only when the number of succeeded pods equals to the completions. When the field is specified, it must be immutable and works only for the Indexed Jobs. Once the Job meets the SuccessPolicy, the lingering pods are terminated.\n\nThis field is beta-level. To use this field, you must enable the `JobSuccessPolicy` feature gate (enabled by default).",
	"backoffLimit":            "Specifies the number of retries before marking this job failed. Defaults to 6",
	"backoffLimitPerIndex":    "Specifies the limit for the number of retries within an index before marking this index as failed. When enabled the number of failures per index is kept in the pod's batch.kubernetes.io/job-index-failure-count annotation. It can only be set when Job's completionMode=Indexed, and the Pod's restart policy is Never. The field is immutable. This field is beta-level. It can be used when the `JobBackoffLimitPerIndex` feature gate is enabled (enabled by default).",
	"maxFailedIndexes":        "Specifies the maximal number of failed indexes before marking the Job as failed, when backoffLimitPerIndex is set. Once the number of failed indexes exceeds this number the entire Job is marked as Failed and its execution is terminated. When left as null the job continues execution of all of its indexes and is marked with the `Complete` Job condition. It can only be specified when backoffLimitPerIndex is set. It can be null or up to completions. It is required and must be less than or equal to 10^4 when is completions greater than 10^5. This field is beta-level. It can be used when the `JobBackoffLimitPerIndex` feature gate is enabled (enabled by default).",
	"selector":                "A label query over pods that should match the pod count. Normally, the system sets this field for you. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors",
	"manualSelector":          "manualSelector controls generation of pod labels and pod selectors. Leave `manualSelector` unset unless you are certain what you are doing. When false or unset, the system pick labels unique to this job and appends those labels to the pod template.  When true, the user is responsible for picking unique labels and specifying the selector.  Failure to pick a unique label may cause this and other jobs to not function correctly.  However, You may see `manualSelector=true` in jobs that were created with the old `extensions/v1beta1` API. More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/#specifying-your-own-pod-selector",
	"template":                "Describes the pod that will be created when executing a job. The only allowed template.spec.restartPolicy values are \"Never\" or \"OnFailure\". More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/",
	"ttlSecondsAfterFinished": "ttlSecondsAfterFinished limits the lifetime of a Job that has finished execution (either Complete or Failed). If this field is set, ttlSecondsAfterFinished after the Job finishes, it is eligible to be automatically deleted. When the Job is being deleted, its lifecycle guarantees (e.g. finalizers) will be honored. If this field is unset, the Job won't be automatically deleted. If this field is set to zero, the Job becomes eligible to be deleted immediately after it finishes.",
	"completionMode":          "completionMode specifies how Pod completions are tracked. It can be `NonIndexed` (default) or `Indexed`.\n\n`NonIndexed` means that the Job is considered complete when there have been .spec.completions successfully completed Pods. Each Pod completion is homologous to each other.\n\n`Indexed` means that the Pods of a Job get an associated completion index from 0 to (.spec.completions - 1), available in the annotation batch.kubernetes.io/job-completion-index. The Job is considered complete when there is one successfully completed Pod for each index. When value is `Indexed`, .spec.completions must be specified and `.spec.parallelism` must be less than or equal to 10^5. In addition, The Pod name takes the form `$(job-name)-$(index)-$(random-string)`, the Pod hostname takes the form `$(job-name)-$(index)`.\n\nMore completion modes can be added in the future. If the Job controller observes a mode that it doesn't recognize, which is possible during upgrades due to version skew, the controller skips updates for the Job.",
	"suspend":                 "suspend specifies whether the Job controller should create Pods or not. If a Job is created with suspend set to true, no Pods are created by the Job controller. If a Job is suspended after creation (i.e. the flag goes from false to true), the Job controller will delete all active Pods associated with this Job. Users must design their workload to gracefully handle this. Suspending a Job will reset the StartTime field of the Job, effectively resetting the ActiveDeadlineSeconds timer too. Defaults to false.",
	"podReplacementPolicy":    "podReplacementPolicy specifies when to create replacement Pods. Possible values are: - TerminatingOrFailed means that we recreate pods\n  when they are terminating (has a metadata.deletionTimestamp) or failed.\n- Failed means to wait until a previously created Pod is fully terminated (has phase\n  Failed or Succeeded) before creating a replacement Pod.\n\nWhen using podFailurePolicy, Failed is the the only allowed value. TerminatingOrFailed and Failed are allowed values when podFailurePolicy is not in use. This is an beta field. To use this, enable the JobPodReplacementPolicy feature toggle. This is on by default.",
	"managedBy":               "ManagedBy field indicates the controller that manages a Job. The k8s Job controller reconciles jobs which don't have this field at all or the field value is the reserved string `kubernetes.io/job-controller`, but skips reconciling Jobs with a custom value for this field. The value must be a valid domain-prefixed path (e.g. acme.io/foo) - all characters before the first \"/\" must be a valid subdomain as defined by RFC 1123. All characters trailing the first \"/\" must be valid HTTP Path characters as defined by RFC 3986. The value cannot exceed 63 characters. This field is immutable.\n\nThis field is alpha-level. The job controller accepts setting the field when the feature gate JobManagedBy is enabled (disabled by default).",
}

func (JobSpec) SwaggerDoc() map[string]string {
	return map_JobSpec
}

var map_JobStatus = map[string]string{
	"":                        "JobStatus represents the current state of a Job.",
	"conditions":              "The latest available observations of an object's current state. When a Job fails, one of the conditions will have type \"Failed\" and status true. When a Job is suspended, one of the conditions will have type \"Suspended\" and status true; when the Job is resumed, the status of this condition will become false. When a Job is completed, one of the conditions will have type \"Complete\" and status true.\n\nA job is considered finished when it is in a terminal condition, either \"Complete\" or \"Failed\". A Job cannot have both the \"Complete\" and \"Failed\" conditions. Additionally, it cannot be in the \"Complete\" and \"FailureTarget\" conditions. The \"Complete\", \"Failed\" and \"FailureTarget\" conditions cannot be disabled.\n\nMore info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/",
	"startTime":               "Represents time when the job controller started processing a job. When a Job is created in the suspended state, this field is not set until the first time it is resumed. This field is reset every time a Job is resumed from suspension. It is represented in RFC3339 form and is in UTC.\n\nOnce set, the field can only be removed when the job is suspended. The field cannot be modified while the job is unsuspended or finished.",
	"completionTime":          "Represents time when the job was completed. It is not guaranteed to be set in happens-before order across separate operations. It is represented in RFC3339 form and is in UTC. The completion time is set when the job finishes successfully, and only then. The value cannot be updated or removed. The value indicates the same or later point in time as the startTime field.",
	"active":                  "The number of pending and running pods which are not terminating (without a deletionTimestamp). The value is zero for finished jobs.",
	"succeeded":               "The number of pods which reached phase Succeeded. The value increases monotonically for a given spec. However, it may decrease in reaction to scale down of elastic indexed jobs.",
	"failed":                  "The number of pods which reached phase Failed. The value increases monotonically.",
	"terminating":             "The number of pods which are terminating (in phase Pending or Running and have a deletionTimestamp).\n\nThis field is beta-level. The job controller populates the field when the feature gate JobPodReplacementPolicy is enabled (enabled by default).",
	"completedIndexes":        "completedIndexes holds the completed indexes when .spec.completionMode = \"Indexed\" in a text format. The indexes are represented as decimal integers separated by commas. The numbers are listed in increasing order. Three or more consecutive numbers are compressed and represented by the first and last element of the series, separated by a hyphen. For example, if the completed indexes are 1, 3, 4, 5 and 7, they are represented as \"1,3-5,7\".",
	"failedIndexes":           "FailedIndexes holds the failed indexes when spec.backoffLimitPerIndex is set. The indexes are represented in the text format analogous as for the `completedIndexes` field, ie. they are kept as decimal integers separated by commas. The numbers are listed in increasing order. Three or more consecutive numbers are compressed and represented by the first and last element of the series, separated by a hyphen. For example, if the failed indexes are 1, 3, 4, 5 and 7, they are represented as \"1,3-5,7\". The set of failed indexes cannot overlap with the set of completed indexes.\n\nThis field is beta-level. It can be used when the `JobBackoffLimitPerIndex` feature gate is enabled (enabled by default).",
	"uncountedTerminatedPods": "uncountedTerminatedPods holds the UIDs of Pods that have terminated but the job controller hasn't yet accounted for in the status counters.\n\nThe job controller creates pods with a finalizer. When a pod terminates (succeeded or failed), the controller does three steps to account for it in the job status:\n\n1. Add the pod UID to the arrays in this field. 2. Remove the pod finalizer. 3. Remove the pod UID from the arrays while increasing the corresponding\n    counter.\n\nOld jobs might not be tracked using this field, in which case the field remains null. The structure is empty for finished jobs.",
	"ready":                   "The number of active pods which have a Ready condition and are not terminating (without a deletionTimestamp).",
}

func (JobStatus) SwaggerDoc() map[string]string {
	return map_JobStatus
}

var map_JobTemplateSpec = map[string]string{
	"":         "JobTemplateSpec describes the data a Job should have when created from a template",
	"metadata": "Standard object's metadata of the jobs created from this template. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata",
	"spec":     "Specification of the desired behavior of the job. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status",
}

func (JobTemplateSpec) SwaggerDoc() map[string]string {
	return map_JobTemplateSpec
}

var map_PodFailurePolicy = map[string]string{
	"":      "PodFailurePolicy describes how failed pods influence the backoffLimit.",
	"rules": "A list of pod failure policy rules. The rules are evaluated in order. Once a rule matches a Pod failure, the remaining of the rules are ignored. When no rule matches the Pod failure, the default handling applies - the counter of pod failures is incremented and it is checked against the backoffLimit. At most 20 elements are allowed.",
}

func (PodFailurePolicy) SwaggerDoc() map[string]string {
	return map_PodFailurePolicy
}

var map_PodFailurePolicyOnExitCodesRequirement = map[string]string{
	"":              "PodFailurePolicyOnExitCodesRequirement describes the requirement for handling a failed pod based on its container exit codes. In particular, it lookups the .state.terminated.exitCode for each app container and init container status, represented by the .status.containerStatuses and .status.initContainerStatuses fields in the Pod status, respectively. Containers completed with success (exit code 0) are excluded from the requirement check.",
	"containerName": "Restricts the check for exit codes to the container with the specified name. When null, the rule applies to all containers. When specified, it should match one the container or initContainer names in the pod template.",
	"operator":      "Represents the relationship between the container exit code(s) and the specified values. Containers completed with success (exit code 0) are excluded from the requirement check. Possible values are:\n\n- In: the requirement is satisfied if at least one container exit code\n  (might be multiple if there are multiple containers not restricted\n  by the 'containerName' field) is in the set of specified values.\n- NotIn: the requirement is satisfied if at least one container exit code\n  (might be multiple if there are multiple containers not restricted\n  by the 'containerName' field) is not in the set of specified values.\nAdditional values are considered to be added in the future. Clients should react to an unknown operator by assuming the requirement is not satisfied.",
	"values":        "Specifies the set of values. Each returned container exit code (might be multiple in case of multiple containers) is checked against this set of values with respect to the operator. The list of values must be ordered and must not contain duplicates. Value '0' cannot be used for the In operator. At least one element is required. At most 255 elements are allowed.",
}

func (PodFailurePolicyOnExitCodesRequirement) SwaggerDoc() map[string]string {
	return map_PodFailurePolicyOnExitCodesRequirement
}

var map_PodFailurePolicyOnPodConditionsPattern = map[string]string{
	"":       "PodFailurePolicyOnPodConditionsPattern describes a pattern for matching an actual pod condition type.",
	"type":   "Specifies the required Pod condition type. To match a pod condition it is required that specified type equals the pod condition type.",
	"status": "Specifies the required Pod condition status. To match a pod condition it is required that the specified status equals the pod condition status. Defaults to True.",
}

func (PodFailurePolicyOnPodConditionsPattern) SwaggerDoc() map[string]string {
	return map_PodFailurePolicyOnPodConditionsPattern
}

var map_PodFailurePolicyRule = map[string]string{
	"":                "PodFailurePolicyRule describes how a pod failure is handled when the requirements are met. One of onExitCodes and onPodConditions, but not both, can be used in each rule.",
	"action":          "Specifies the action taken on a pod failure when the requirements are satisfied. Possible values are:\n\n- FailJob: indicates that the pod's job is marked as Failed and all\n  running pods are terminated.\n- FailIndex: indicates that the pod's index is marked as Failed and will\n  not be restarted.\n  This value is beta-level. It can be used when the\n  `JobBackoffLimitPerIndex` feature gate is enabled (enabled by default).\n- Ignore: indicates that the counter towards the .backoffLimit is not\n  incremented and a replacement pod is created.\n- Count: indicates that the pod is handled in the default way - the\n  counter towards the .backoffLimit is incremented.\nAdditional values are considered to be added in the future. Clients should react to an unknown action by skipping the rule.",
	"onExitCodes":     "Represents the requirement on the container exit codes.",
	"onPodConditions": "Represents the requirement on the pod conditions. The requirement is represented as a list of pod condition patterns. The requirement is satisfied if at least one pattern matches an actual pod condition. At most 20 elements are allowed.",
}

func (PodFailurePolicyRule) SwaggerDoc() map[string]string {
	return map_PodFailurePolicyRule
}

var map_SuccessPolicy = map[string]string{
	"":      "SuccessPolicy describes when a Job can be declared as succeeded based on the success of some indexes.",
	"rules": "rules represents the list of alternative rules for the declaring the Jobs as successful before `.status.succeeded >= .spec.completions`. Once any of the rules are met, the \"SucceededCriteriaMet\" condition is added, and the lingering pods are removed. The terminal state for such a Job has the \"Complete\" condition. Additionally, these rules are evaluated in order; Once the Job meets one of the rules, other rules are ignored. At most 20 elements are allowed.",
}

func (SuccessPolicy) SwaggerDoc() map[string]string {
	return map_SuccessPolicy
}

var map_SuccessPolicyRule = map[string]string{
	"":                 "SuccessPolicyRule describes rule for declaring a Job as succeeded. Each rule must have at least one of the \"succeededIndexes\" or \"succeededCount\" specified.",
	"succeededIndexes": "succeededIndexes specifies the set of indexes which need to be contained in the actual set of the succeeded indexes for the Job. The list of indexes must be within 0 to \".spec.completions-1\" and must not contain duplicates. At least one element is required. The indexes are represented as intervals separated by commas. The intervals can be a decimal integer or a pair of decimal integers separated by a hyphen. The number are listed in represented by the first and last element of the series, separated by a hyphen. For example, if the completed indexes are 1, 3, 4, 5 and 7, they are represented as \"1,3-5,7\". When this field is null, this field doesn't default to any value and is never evaluated at any time.",
	"succeededCount":   "succeededCount specifies the minimal required size of the actual set of the succeeded indexes for the Job. When succeededCount is used along with succeededIndexes, the check is constrained only to the set of indexes specified by succeededIndexes. For example, given that succeededIndexes is \"1-4\", succeededCount is \"3\", and completed indexes are \"1\", \"3\", and \"5\", the Job isn't declared as succeeded because only \"1\" and \"3\" indexes are considered in that rules. When this field is null, this doesn't default to any value and is never evaluated at any time. When specified it needs to be a positive integer.",
}

func (SuccessPolicyRule) SwaggerDoc() map[string]string {
	return map_SuccessPolicyRule
}

var map_UncountedTerminatedPods = map[string]string{
	"":          "UncountedTerminatedPods holds UIDs of Pods that have terminated but haven't been accounted in Job status counters.",
	"succeeded": "succeeded holds UIDs of succeeded Pods.",
	"failed":    "failed holds UIDs of failed Pods.",
}

func (UncountedTerminatedPods) SwaggerDoc() map[string]string {
	return map_UncountedTerminatedPods
}

// AUTO-GENERATED FUNCTIONS END HERE
