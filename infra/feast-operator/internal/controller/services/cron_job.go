package services

import (
	"os"

	feastdevv1alpha1 "github.com/feast-dev/feast/infra/feast-operator/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (feast *FeastServices) createFeastRole() error {
	logger := log.FromContext(feast.Handler.Context)
	role := feast.initFeastRole()
	if op, err := controllerutil.CreateOrUpdate(feast.Handler.Context, feast.Handler.Client, role, controllerutil.MutateFn(func() error {
		return feast.setFeastRole(role)
	})); err != nil {
		return err
	} else if op == controllerutil.OperationResultCreated || op == controllerutil.OperationResultUpdated {
		logger.Info("Successfully reconciled", "Role", role.Name, "operation", op)
	}

	return nil
}

func (feast *FeastServices) initFeastRole() *rbacv1.Role {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: feast.getFeastRoleName(), Namespace: feast.Handler.FeatureStore.Namespace},
	}
	role.SetGroupVersionKind(rbacv1.SchemeGroupVersion.WithKind("Role"))
	return role
}

func (feast *FeastServices) setFeastRole(role *rbacv1.Role) error {
	role.Labels = feast.getLabels()
	role.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{rbacv1.GroupName},
			Resources: []string{"roles", "rolebindings"},
			Verbs:     []string{"get", "list", "watch"},
		},
	}

	return controllerutil.SetControllerReference(feast.Handler.FeatureStore, role, feast.Handler.Scheme)
}

func (feast *FeastServices) createFeastRoleBinding() error {
	logger := log.FromContext(feast.Handler.Context)
	roleBinding := feast.initFeastRoleBinding()
	if op, err := controllerutil.CreateOrUpdate(feast.Handler.Context, feast.Handler.Client, roleBinding, controllerutil.MutateFn(func() error {
		return feast.setFeastRoleBinding(roleBinding)
	})); err != nil {
		return err
	} else if op == controllerutil.OperationResultCreated || op == controllerutil.OperationResultUpdated {
		logger.Info("Successfully reconciled", "RoleBinding", roleBinding.Name, "operation", op)
	}

	return nil
}

func setDefaultCronJobConfigs(feastCronJob *feastdevv1alpha1.FeastCronJob) {
	if feastCronJob == nil {
		feastCronJob = &feastdevv1alpha1.FeastCronJob{}
	}
	if len(feastCronJob.Schedule) == 0 {
		feastCronJob.Schedule = "@yearly"
		if feastCronJob.Suspend == nil {
			feastCronJob.Suspend = boolPtr(true)
		}
		if len(feastCronJob.ConcurrencyPolicy) == 0 {
			feastCronJob.ConcurrencyPolicy = batchv1.ReplaceConcurrent
		}
		if feastCronJob.StartingDeadlineSeconds == nil {
			feastCronJob.StartingDeadlineSeconds = int64Ptr(5)
		}
	}

	ctrCfgs := feastCronJob.ContainerConfigs
	if ctrCfgs == nil {
		ctrCfgs = &feastdevv1alpha1.JobContainerConfigs{}
	}
	if ctrCfgs.Image == nil {
		img := getCronJobImage()
		ctrCfgs.Image = &img
	}
	if len(ctrCfgs.Commands) == 0 {
		ctrCfgs.Commands = []string{
			"feast apply",
			"feast materialize-incremental $(date -u +'%Y-%m-%dT%H:%M:%S')",
		}
	}
}

/*
	jobTemplate:
	  spec:
	    template:
	      spec:
	        serviceAccountName: feast-sample
	        initContainers:
	        - name: apply
	          image: quay.io/openshift/origin-cli:4.17
	          command:
	          - "kubectl"
	          - "exec"
	          - "deploy/feast-sample"
	          - "-ic"
	          - "feast"
	          - "--"
	          - "bash"
	          - "-c"
	          - "feast apply"
	        containers:
	        - name: materialize
	          image: quay.io/openshift/origin-cli:4.17
	          command:
	          - "kubectl"
	          - "exec"
	          - "deploy/feast-sample"
	          - "-ic"
	          - "feast"
	          - "--"
	          - "bash"
	          - "-c"
	          - 'feast materialize-incremental $(date -u +"%Y-%m-%dT%H:%M:%S")'

####          restartPolicy: Never
*/

func getCronJobImage() string {
	if img, exists := os.LookupEnv(cronJobImageVar); exists {
		return img
	}
	return DefaultCronJobImage
}
