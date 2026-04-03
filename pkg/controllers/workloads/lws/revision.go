package lws

import (
	"context"
	"encoding/json"
	"fmt"
	"hash"
	"hash/fnv"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/dump"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	leaderworkersetv1 "sigs.k8s.io/lws/api/leaderworkerset/v1"
)

func GetRevisionKey(obj metav1.Object) string {
	if obj.GetLabels() != nil {
		return obj.GetLabels()[leaderworkersetv1.RevisionKey]
	}
	return ""
}

func GetOrCreateRevision(ctx context.Context, k8sClient client.Client, lws *leaderworkersetv1.LeaderWorkerSet, revisionKey string) (*appsv1.ControllerRevision, error) {
	// 1. get revision by revisionKey
	revision, err := GetRevision(ctx, k8sClient, lws, revisionKey)
	if err != nil {
		return nil, err
	}

	if revision != nil {
		return revision, nil
	}

	// 2.create revision
	revision, err = NewRevision(ctx, k8sClient, lws, revisionKey)
	if err != nil {
		klog.Errorf("new revision object error: %v", err)
		return nil, err
	}

	err = k8sClient.Create(ctx, revision)
	if err != nil {
		klog.Errorf("k8s create revision error: %v", err)
		return nil, err
	}

	return revision, nil
}

func NewRevision(ctx context.Context, k8sClient client.Client, lws *leaderworkersetv1.LeaderWorkerSet, revisionKey string) (*appsv1.ControllerRevision, error) {
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: map[string]string{
		leaderworkersetv1.SetNameLabelKey: lws.Name,
	}})
	if err != nil {
		return nil, err
	}

	revisions, err := ListRevisions(ctx, k8sClient, lws, selector)
	if err != nil {
		klog.Errorf("Listing all controller revisions error: %v", err)
		return nil, err
	}

	revisionNumber := int64(1)
	highestRevision := getHighestRevision(revisions)
	if highestRevision != nil {
		revisionNumber = highestRevision.Revision + 1
	}

	patch, err := getRevisionPatchForLWS(lws)
	if err != nil {
		klog.Errorf("get revision patch error: %v", err)
		return nil, err
	}

	revision := &appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: lws.Namespace,
			Labels: map[string]string{
				leaderworkersetv1.SetNameLabelKey: lws.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(lws, leaderworkersetv1.SchemeGroupVersion.WithKind(lws.Kind)),
			},
		},
		Revision: revisionNumber,
		Data:     runtime.RawExtension{Raw: patch},
	}

	hashRevision := HashRevision(revision)
	if revisionKey == "" {
		revisionKey = hashRevision
	}

	revision.Name = fmt.Sprintf("%s-%s-%v", lws.Name, hashRevision, revisionNumber)
	revision.Labels[leaderworkersetv1.RevisionKey] = revisionKey

	return revision, nil
}

func HashRevision(revision *appsv1.ControllerRevision) string {
	hasher := fnv.New32()
	if len(revision.Data.Raw) > 0 {
		hasher.Write(revision.Data.Raw)
	}
	if revision.Data.Object != nil {
		deepHashObject(hasher, revision.Data.Object)
	}

	return rand.SafeEncodeString(fmt.Sprint(hasher.Sum32()))
}

func deepHashObject(hasher hash.Hash, objectToWrite interface{}) {
	hasher.Reset()
	fmt.Fprintf(hasher, "%v", dump.ForHash(objectToWrite))
}

func getRevisionPatchForLWS(lws *leaderworkersetv1.LeaderWorkerSet) ([]byte, error) {
	clone := lws.DeepCopy()

	if clone.Spec.NetworkConfig == nil {
		subdomainPolicy := leaderworkersetv1.SubdomainShared
		clone.Spec.NetworkConfig = &leaderworkersetv1.NetworkConfig{
			SubdomainPolicy: &subdomainPolicy,
		}
	}

	/*str := &bytes.Buffer{}
	if err := unstructured.UnstructuredJSONScheme.Encode(clone, str); err != nil {
		return nil, err
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(str.Bytes(), &raw); err != nil {
		return nil, err
	}*/

	raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(clone)
	if err != nil {
		return nil, err
	}

	specCopy := make(map[string]interface{})
	objCopy := make(map[string]interface{})
	spec := raw["spec"].(map[string]interface{})
	networkConfig := spec["networkConfig"].(map[string]interface{})
	networkConfig["$patch"] = "replace"
	specCopy["networkConfig"] = networkConfig
	template := spec["leaderWorkerTemplate"].(map[string]interface{})
	template["$patch"] = "replace"
	specCopy["leaderWorkerTemplate"] = template
	objCopy["spec"] = specCopy

	return json.Marshal(objCopy)
}

func GetRevision(ctx context.Context, k8sClient client.Client, lws *leaderworkersetv1.LeaderWorkerSet, revisionKey string) (*appsv1.ControllerRevision, error) {
	if revisionKey == "" {
		return nil, nil
	}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: map[string]string{
		leaderworkersetv1.SetNameLabelKey: lws.Name,
		leaderworkersetv1.RevisionKey:     revisionKey,
	}})
	if err != nil {
		return nil, err
	}

	revisions, err := ListRevisions(ctx, k8sClient, lws, selector)
	if err != nil {
		klog.Errorf("Listing all controller revisions error: %v", err)
		return nil, err
	}

	if len(revisions) == 0 {
		return nil, nil
	}

	if len(revisions) > 1 {
		// Since we only create a controllerRevision when the template hash changes, only one should match
		klog.Errorf("More than one revision exists for the given template hash; returning the latest revision error: %v", err)
		return getHighestRevision(revisions), nil
	}

	return revisions[0], nil
}

func ListRevisions(ctx context.Context, k8sClient client.Client, obj metav1.Object, selector labels.Selector) ([]*appsv1.ControllerRevision, error) {
	// List all revisions in the namespace that match the selector
	// k get controllerrevisions -n lws-system -l leaderworkerset.sigs.k8s.io/name=lws-min1,leaderworkerset.sigs.k8s.io/template-revision-hash=7dd9b94dfc
	revisionList := new(appsv1.ControllerRevisionList)
	err := k8sClient.List(ctx, revisionList, client.InNamespace(obj.GetNamespace()), client.MatchingLabelsSelector{Selector: selector})
	if err != nil {
		return nil, err
	}

	var revisions []*appsv1.ControllerRevision
	for i := range revisionList.Items {
		revision := &revisionList.Items[i]
		for _, ownerReference := range revision.OwnerReferences {
			if ownerReference.UID == obj.GetUID() {
				revisions = append(revisions, revision)
			}
		}
	}

	return revisions, nil
}

func getHighestRevision(revisions []*appsv1.ControllerRevision) *appsv1.ControllerRevision {
	count := len(revisions)
	if count <= 0 {
		return nil
	}

	start := int64(0)
	var maxRevision *appsv1.ControllerRevision
	for _, revision := range revisions {
		if start <= revision.Revision {
			start = revision.Revision
			maxRevision = revision
		}
	}
	return maxRevision
}

// ApplyRevision 根据最新的 revision 回滚生成上一版本的 original lws
func ApplyRevision(lws *leaderworkersetv1.LeaderWorkerSet, revision *appsv1.ControllerRevision) (*leaderworkersetv1.LeaderWorkerSet, error) {
	clone := lws.DeepCopy()
	cloneBytes, err := json.Marshal( clone)
	/*str := &bytes.Buffer{}
	err := unstructured.UnstructuredJSONScheme.Encode(clone, str)
	if err != nil {
		return nil, err
	}*/
	patched, err := strategicpatch.StrategicMergePatch(cloneBytes, revision.Data.Raw, clone)
	if err != nil {
		return nil, err
	}
	restoredLws := &leaderworkersetv1.LeaderWorkerSet{}
	if err = json.Unmarshal(patched, restoredLws); err != nil {
		return nil, err
	}

	return restoredLws, nil
}
