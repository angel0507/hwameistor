// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/hwameistor/hwameistor/pkg/apis/hwameistor/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// LocalVolumeExpandLister helps list LocalVolumeExpands.
// All objects returned here must be treated as read-only.
type LocalVolumeExpandLister interface {
	// List lists all LocalVolumeExpands in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.LocalVolumeExpand, err error)
	// Get retrieves the LocalVolumeExpand from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.LocalVolumeExpand, error)
	LocalVolumeExpandListerExpansion
}

// localVolumeExpandLister implements the LocalVolumeExpandLister interface.
type localVolumeExpandLister struct {
	indexer cache.Indexer
}

// NewLocalVolumeExpandLister returns a new LocalVolumeExpandLister.
func NewLocalVolumeExpandLister(indexer cache.Indexer) LocalVolumeExpandLister {
	return &localVolumeExpandLister{indexer: indexer}
}

// List lists all LocalVolumeExpands in the indexer.
func (s *localVolumeExpandLister) List(selector labels.Selector) (ret []*v1alpha1.LocalVolumeExpand, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.LocalVolumeExpand))
	})
	return ret, err
}

// Get retrieves the LocalVolumeExpand from the index for a given name.
func (s *localVolumeExpandLister) Get(name string) (*v1alpha1.LocalVolumeExpand, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("localvolumeexpand"), name)
	}
	return obj.(*v1alpha1.LocalVolumeExpand), nil
}
