package cache

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

func (c *cache) addPod(obj interface{}) {
	pod := obj.(*corev1.Pod)
	modelName, ok := getModelNameFromPod(pod)
	if !ok {
		klog.V(4).InfoS("ignored pod without model label or annotation", "name", pod.Name)
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.updateModels(modelName, pod)

}

func (c *cache) updatePod(oldObj interface{}, newObj interface{}) {

}

func (c *cache) deletePod(obj interface{}) {

}

func (c *cache) updateModels(modelName string, pod *corev1.Pod) {
	c.models.LoadOrStore(modelName, pod)
}

func getModelNameFromPod(pod *corev1.Pod) (string, bool) {
	// Try label first (standard case)
	if modelName, ok := pod.Labels[modelIdentifier]; ok && modelName != "" {
		return modelName, true
	}
	// Fallback to annotation (allows special characters like '/' in model paths)
	if modelName, ok := pod.Annotations[modelIdentifier]; ok && modelName != "" {
		return modelName, true
	}
	return "", false
}
