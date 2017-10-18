package vm

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/jeevatkm/go-model"
	"k8s.io/apimachinery/pkg/api/errors"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	genericvalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/api/validation/path"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/endpoints/request"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
	storeerr "k8s.io/apiserver/pkg/storage/errors"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
)

/*
"fmt"
"time"
"reflect"
"strings"

genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
genericvalidation "k8s.io/apimachinery/pkg/api/validation"
"github.com/golang/glog"
"k8s.io/apimachinery/pkg/api/errors"
"k8s.io/apimachinery/pkg/api/meta"
"k8s.io/apimachinery/pkg/fields"
"k8s.io/apimachinery/pkg/labels"
"k8s.io/apimachinery/pkg/runtime"
"k8s.io/apimachinery/pkg/runtime/schema"
"k8s.io/apimachinery/pkg/util/validation/field"
"k8s.io/apimachinery/pkg/watch"
"k8s.io/apiserver/pkg/endpoints/request"
"k8s.io/apiserver/pkg/registry/generic"
"k8s.io/apiserver/pkg/registry/rest"
"k8s.io/apiserver/pkg/storage"
kubeerr "k8s.io/apimachinery/pkg/api/errors"
metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
storeerr "k8s.io/apiserver/pkg/storage/errors"
utilruntime "k8s.io/apimachinery/pkg/util/runtime"
metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
"k8s.io/apimachinery/pkg/util/sets"
"github.com/jeevatkm/go-model"
"k8s.io/apimachinery/pkg/api/validation/path"

*/

type REST struct {
	*genericregistry.Store
}

type VMCopier struct {
}

var errEmptiedFinalizers = fmt.Errorf("emptied finalizers")

// NewREST returns a RESTStorage object that will work against API services.
func NewREST(scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter) (*REST, error) {
	strategy := NewStrategy(scheme)
	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}

	/*groupResource := schema.GroupResource{Group: v1.GroupName, Resource: "VirtualMachine"}
	//ropt, _ := options.RESTOptions.GetRESTOptions(scheme)
	ropt, err := optsGetter.GetRESTOptions(groupResource)
	if err != nil {
		return nil, err
	}
	prefix := ropt.ResourcePrefix*/

	store := &genericregistry.Store{
		//Copier:                   VMCopier{},
		Copier:                   scheme,
		NewFunc:                  func() runtime.Object { return &v1.VirtualMachine{} },
		NewListFunc:              func() runtime.Object { return &v1.VirtualMachineList{} },
		PredicateFunc:            MatchVM,
		DefaultQualifiedResource: v1.Resource("virtualmachines"),

		CreateStrategy: strategy,
		UpdateStrategy: strategy,
		DeleteStrategy: strategy,
		//KeyFunc:        getKeyFunc(prefix),
		//KeyRootFunc:    getKeyRootFunc(prefix),
	}

	if err := store.CompleteWithOptions(options); err != nil {
		logging.DefaultLogger().Error().Reason(err).Msg("Unable to create REST storage for vm resource")
		return nil, fmt.Errorf("Unable to create REST storage for vm resource: %v.", err)
	}
	return &REST{store}, nil
}

//FIXME: this might not be needed. try omitting it once everything else works
func (VMCopier) Copy(obj runtime.Object) (runtime.Object, error) {
	//if objCopy := obj.DeepCopyObject(); objCopy != nil {
	//	return objCopy, nil
	//} else {
	//	return nil, nil
	//}

	log := logging.DefaultLogger()
	log.Info().Msgf("inside VM Copy function")
	log.Info().Msgf("Source object: %v", obj)
	log.Info().Msgf("Source Object Kind: %v", obj.GetObjectKind())

	vm := v1.VirtualMachine{}
	model.Copy(&vm, &obj)

	log.Info().Msgf("Copied object: %v", vm)
	log.Info().Msgf("Copied Object Name: %v", vm.GetObjectMeta().GetName())
	log.Info().Msgf("Copied Object Namespace: %v", vm.GetObjectMeta().GetNamespace())
	log.Info().Msgf("Copied Object UID: %v", vm.GetObjectMeta().GetUID())

	return &vm, nil
	//objCopy := vm.DeepCopyObject()
	//return objCopy, nil

	//objCopy := v1.VirtualMachine{}
	//objCopy.DeepCopyObject()
	//errs := model.Copy(&objCopy, &obj)
	//if len(errs) != 0 {
	//	return nil, errors.NewAggregate(errs)
	//}
	//
	//
	//return runtime.Object(objCopy), nil
}

var (
	errAlreadyDeleting = fmt.Errorf("abort delete")
	errDeleteNow       = fmt.Errorf("delete now")
	//errEmptiedFinalizers = fmt.Errorf("emptied finalizers")
)

func getKeyRootFunc(prefix string) func(genericapirequest.Context) string {
	logging.DefaultLogger().Info().Msgf("Creating keyRootFunc. path prefix: %s", prefix)
	return func(ctx genericapirequest.Context) string {
		return keyRootFunc(ctx, prefix)
	}
}

func getKeyFunc(prefix string) func(ctx genericapirequest.Context, name string) (string, error) {
	logging.DefaultLogger().Info().Msgf("Creating keyFunc. path prefix: %s", prefix)
	return func(ctx genericapirequest.Context, name string) (string, error) {
		return keyFunc(ctx, prefix, name)
	}
}

func keyRootFunc(ctx genericapirequest.Context, prefix string) string {
	logging.DefaultLogger().Info().Msgf("keyRootFunc called.")
	key := prefix
	ns, ok := genericapirequest.NamespaceFrom(ctx)
	if ok && len(ns) > 0 {
		key = key + "/" + ns
	}
	logging.DefaultLogger().Info().Msgf("keyRootFunc result: %s", key)
	return key
}

func keyFunc(ctx genericapirequest.Context, prefix string, name string) (string, error) {
	logging.DefaultLogger().Info().Msgf("keyFunc called.")
	key := keyRootFunc(ctx, prefix)
	ns, ok := genericapirequest.NamespaceFrom(ctx)
	if !ok || len(ns) == 0 {
		ns = "default"
	}
	if len(name) == 0 {
		return "", kubeerr.NewBadRequest("Name parameter required.")
	}
	if msgs := path.IsValidPathSegmentName(name); len(msgs) != 0 {
		return "", kubeerr.NewBadRequest(fmt.Sprintf("Name parameter invalid: %q: %s", name, strings.Join(msgs, ";")))
	}
	key = key + "/" + name
	logging.DefaultLogger().Info().Msgf("keyFunc result: %s", key)
	return key, nil
}

// BeforeCreate ensures that common operations for all resources are performed on creation. It only returns
// errors that can be converted to api.Status. It invokes PrepareForCreate, then GenerateName, then Validate.
// It returns nil if the object should be created.
func BeforeCreate(strategy rest.RESTCreateStrategy, ctx genericapirequest.Context, obj runtime.Object) error {
	log := logging.DefaultLogger()
	log.Info().Msgf("Inside custom BeforeCreate call")
	objectMeta, kind, kerr := objectMetaAndKind(strategy, obj)
	if kerr != nil {
		return kerr
	}

	if strategy.NamespaceScoped() {
		if !rest.ValidNamespace(ctx, objectMeta) {
			return errors.NewBadRequest("the namespace of the provided object does not match the namespace sent on the request")
		}
	} else {
		objectMeta.SetNamespace(metav1.NamespaceNone)
	}

	objectMeta.SetDeletionTimestamp(nil)
	objectMeta.SetDeletionGracePeriodSeconds(nil)
	strategy.PrepareForCreate(ctx, obj)
	rest.FillObjectMetaSystemFields(ctx, objectMeta)
	if len(objectMeta.GetGenerateName()) > 0 && len(objectMeta.GetName()) == 0 {
		objectMeta.SetName(strategy.GenerateName(objectMeta.GetGenerateName()))
	}

	// ClusterName is ignored and should not be saved
	objectMeta.SetClusterName("")

	if errs := strategy.Validate(ctx, obj); len(errs) > 0 {
		return errors.NewInvalid(kind.GroupKind(), objectMeta.GetName(), errs)
	}

	log.Info().Msgf("before generic checks")
	// Custom validation (including name validation) passed
	// Now run common validation on object meta
	// Do this *after* custom validation so that specific error messages are shown whenever possible
	if errs := genericvalidation.ValidateObjectMetaAccessor(objectMeta, strategy.NamespaceScoped(), path.ValidatePathSegmentName, field.NewPath("metadata")); len(errs) > 0 {
		return errors.NewInvalid(kind.GroupKind(), objectMeta.GetName(), errs)
	}

	log.Info().Msgf("generic checks succeeded")
	strategy.Canonicalize(obj)
	return nil
}

// Get retrieves the item from storage.
func (e *REST) Get(ctx genericapirequest.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	log := logging.DefaultLogger()
	log.Info().Msgf("Inside custom GET handler")
	log.Info().Msgf("context: %v", ctx)
	log.Info().Msgf("name: %v", ctx)
	log.Info().Msgf("options: %v", options)
	obj := e.NewFunc()
	key, err := e.KeyFunc(ctx, name)
	log.Info().Msgf("key: %v", key)
	if err != nil {
		return nil, err
	}
	if err := e.Storage.Get(ctx, key, options.ResourceVersion, obj, false); err != nil {
		return nil, storeerr.InterpretGetError(err, e.qualifiedResourceFromContext(ctx), name)
	}
	if e.Decorator != nil {
		if err := e.Decorator(obj); err != nil {
			return nil, err
		}
	}
	vm := obj.(*v1.VirtualMachine)
	log.Info().Msgf("VM: %v", vm)
	return obj, nil
}

// Create inserts a new item according to the unique key from the object.
func (e *REST) Create(ctx genericapirequest.Context, obj runtime.Object, includeUninitialized bool) (runtime.Object, error) {
	log := logging.DefaultLogger()
	log.Info().Msgf("Inside custom Create handler")

	vm := obj.(*v1.VirtualMachine)
	log.Info().Msgf("VM: %v", vm)

	if err := BeforeCreate(e.CreateStrategy, ctx, obj); err != nil {
		log.Info().Reason(err).Msgf("Error occured before creation")
		return nil, err
	}

	name, err := e.ObjectNameFunc(obj)
	log.Info().Msgf("Object name: %v", name)
	if err != nil {
		return nil, err
	}

	key, err := e.KeyFunc(ctx, name)
	log.Info().Msgf("Object key: %v", key)
	if err != nil {
		return nil, err
	}

	qualifiedResource := e.qualifiedResourceFromContext(ctx)

	log.Info().Msgf("qualifiedResource: %v", qualifiedResource)

	ttl, err := e.calculateTTL(obj, 0, false)
	log.Info().Msgf("ttl: %v", ttl)

	if err != nil {
		return nil, err
	}

	out := e.NewFunc()
	if err := e.Storage.Create(ctx, key, obj, out, ttl); err != nil {
		err = storeerr.InterpretCreateError(err, qualifiedResource, name)

		err = rest.CheckGeneratedNameError(e.CreateStrategy, err, obj)
		if !kubeerr.IsAlreadyExists(err) {
			return nil, err
		}

		if errGet := e.Storage.Get(ctx, key, "", out, false); errGet != nil {
			return nil, err
		}

		accessor, errGetAcc := meta.Accessor(out)
		if errGetAcc != nil {
			return nil, err
		}

		if accessor.GetDeletionTimestamp() != nil {
			msg := &err.(*kubeerr.StatusError).ErrStatus.Message
			*msg = fmt.Sprintf("object is being deleted: %s", *msg)
		}
		return nil, err
	}

	if e.AfterCreate != nil {
		if err := e.AfterCreate(out); err != nil {
			return nil, err
		}
	}

	if e.Decorator != nil {
		if err := e.Decorator(obj); err != nil {
			return nil, err
		}
	}

	if !includeUninitialized {
		return e.WaitForInitialized(ctx, out)
	}

	return out, nil
}

// Update performs an atomic update and set of the object. Returns the result of the update
// or an error. If the registry allows create-on-update, the create flow will be executed.
// A bool is returned along with the object and any errors, to indicate object creation.
func (e *REST) Update(ctx genericapirequest.Context, name string, objInfo rest.UpdatedObjectInfo) (runtime.Object, bool, error) {
	log := logging.DefaultLogger()
	log.Info().Msgf("Inside custom Update handler")
	key, err := e.KeyFunc(ctx, name)
	if err != nil {
		return nil, false, err
	}
	log.Info().Msgf("key: %v", key)

	var (
		creatingObj runtime.Object
		creating    = false
	)

	qualifiedResource := e.qualifiedResourceFromContext(ctx)
	storagePreconditions := &storage.Preconditions{}
	if preconditions := objInfo.Preconditions(); preconditions != nil {
		storagePreconditions.UID = preconditions.UID
	}

	out := e.NewFunc()
	// deleteObj is only used in case a deletion is carried out
	var deleteObj runtime.Object
	log.Info().Msgf("Calling GuaranteedUpdate")
	err = e.Storage.GuaranteedUpdate(ctx, key, out, true, storagePreconditions, func(existing runtime.Object, res storage.ResponseMeta) (runtime.Object, *uint64, error) {
		log.Info().Msgf("Inside tryUpdate")
		vm := existing.(*v1.VirtualMachine)
		log.Info().Msgf("existing.name: %s", vm.GetObjectMeta().GetName())
		// Given the existing object, get the new object
		obj, err := objInfo.UpdatedObject(ctx, existing)
		if err != nil {
			return nil, nil, err
		}
		log.Info().Msgf("tryUpdate checkpoint 1")
		// If AllowUnconditionalUpdate() is true and the object specified by
		// the user does not have a resource version, then we populate it with
		// the latest version. Else, we check that the version specified by
		// the user matches the version of latest storage object.
		resourceVersion, err := e.Storage.Versioner().ObjectResourceVersion(obj)
		if err != nil {
			return nil, nil, err
		}
		log.Info().Msgf("tryUpdate checkpoint 2")
		doUnconditionalUpdate := resourceVersion == 0 && e.UpdateStrategy.AllowUnconditionalUpdate()

		log.Info().Msgf("tryUpdate checkpoint 3")

		version, err := e.Storage.Versioner().ObjectResourceVersion(existing)
		if err != nil {
			return nil, nil, err
		}
		log.Info().Msgf("tryUpdate checkpoint 4")
		if version == 0 {
			if !e.UpdateStrategy.AllowCreateOnUpdate() {
				return nil, nil, kubeerr.NewNotFound(qualifiedResource, name)
			}
			creating = true
			creatingObj = obj
			if err := rest.BeforeCreate(e.CreateStrategy, ctx, obj); err != nil {
				return nil, nil, err
			}
			ttl, err := e.calculateTTL(obj, 0, false)
			if err != nil {
				return nil, nil, err
			}
			return obj, &ttl, nil
		}
		log.Info().Msgf("tryUpdate checkpoint 5")

		creating = false
		creatingObj = nil
		if doUnconditionalUpdate {
			log.Info().Msgf("unconditional update path")
			// Update the object's resource version to match the latest
			// storage object's resource version.
			err = e.Storage.Versioner().UpdateObject(obj, res.ResourceVersion)
			if err != nil {
				return nil, nil, err
			}
		} else {
			log.Info().Msgf("unconditional update not allowed")
			// Check if the object's resource version matches the latest
			// resource version.
			newVersion, err := e.Storage.Versioner().ObjectResourceVersion(obj)
			if err != nil {
				return nil, nil, err
			}
			if newVersion == 0 {
				// TODO: The Invalid error should have a field for Resource.
				// After that field is added, we should fill the Resource and
				// leave the Kind field empty. See the discussion in #18526.
				qualifiedKind := schema.GroupKind{Group: qualifiedResource.Group, Kind: qualifiedResource.Resource}
				fieldErrList := field.ErrorList{field.Invalid(field.NewPath("metadata").Child("resourceVersion"), newVersion, "must be specified for an update")}
				return nil, nil, kubeerr.NewInvalid(qualifiedKind, name, fieldErrList)
			}
			if newVersion != version {
				return nil, nil, kubeerr.NewConflict(qualifiedResource, name, fmt.Errorf(genericregistry.OptimisticLockErrorMsg))
			}
		}
		log.Info().Msgf("tryUpdate checkpoint 6")
		if err := rest.BeforeUpdate(e.UpdateStrategy, ctx, obj, existing); err != nil {
			return nil, nil, err
		}
		log.Info().Msgf("tryUpdate checkpoint 7")
		if e.shouldDeleteDuringUpdate(ctx, key, obj, existing) {
			deleteObj = obj
			return nil, nil, errEmptiedFinalizers
		}
		log.Info().Msgf("tryUpdate checkpoint 8")
		ttl, err := e.calculateTTL(obj, res.TTL, true)
		if err != nil {
			return nil, nil, err
		}
		log.Info().Msgf("tryUpdate checkpoint 9")
		if int64(ttl) != res.TTL {
			return obj, &ttl, nil
		}
		log.Info().Msgf("tryUpdate checkpoint 10")
		return obj, nil, nil
	})

	if err != nil {
		log.Info().Msgf("update failed.")
		// delete the object
		if err == errEmptiedFinalizers {
			log.Info().Msgf("deleting object.")
			return e.deleteWithoutFinalizers(ctx, name, key, deleteObj, storagePreconditions)
		}
		if creating {
			log.Info().Msgf("failure while creating?")
			err = storeerr.InterpretCreateError(err, qualifiedResource, name)
			err = rest.CheckGeneratedNameError(e.CreateStrategy, err, creatingObj)
		} else {
			log.Info().Msgf("failure after creation")
			err = storeerr.InterpretUpdateError(err, qualifiedResource, name)
		}
		return nil, false, err
	}

	if e.shouldDeleteForFailedInitialization(ctx, out) {
		return e.deleteWithoutFinalizers(ctx, name, key, out, storagePreconditions)
	}

	if creating {
		if e.AfterCreate != nil {
			if err := e.AfterCreate(out); err != nil {
				return nil, false, err
			}
		}
	} else {
		if e.AfterUpdate != nil {
			if err := e.AfterUpdate(out); err != nil {
				return nil, false, err
			}
		}
	}
	if e.Decorator != nil {
		if err := e.Decorator(out); err != nil {
			return nil, false, err
		}
	}
	return out, creating, nil
}

// List returns a list of items matching labels and field according to the
// store's PredicateFunc.
func (e *REST) List(ctx genericapirequest.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	log := logging.DefaultLogger()
	log.Info().Msgf("Inside custom List handler")
	label := labels.Everything()
	log.Info().Msgf("label: %v", label)

	if options != nil && options.LabelSelector != nil {
		label = options.LabelSelector
	}
	field := fields.Everything()
	log.Info().Msgf("field: %v", field)

	if options != nil && options.FieldSelector != nil {
		field = options.FieldSelector
	}
	out, err := e.ListPredicate(ctx, e.PredicateFunc(label, field), options)
	if err != nil {
		return nil, err
	}
	if e.Decorator != nil {
		if err := e.Decorator(out); err != nil {
			return nil, err
		}
	}
	log.Info().Msgf("output: %v", out)
	vmList := out.(*v1.VirtualMachineList)
	for idx, vm := range vmList.Items {
		log.Info().Msgf("VM[%d]: %v", idx, vm)
	}
	log.Info().Msgf("context: %v", ctx)

	return out, nil
}

func (e *REST) Watch(ctx genericapirequest.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	log := logging.DefaultLogger()
	log.Info().Msgf("Inside custom List handler")
	label := labels.Everything()
	log.Info().Msgf("label: %v", label)
	if options != nil && options.LabelSelector != nil {
		label = options.LabelSelector
	}
	field := fields.Everything()
	log.Info().Msgf("field: %v", field)
	if options != nil && options.FieldSelector != nil {
		field = options.FieldSelector
	}
	predicate := e.PredicateFunc(label, field)

	resourceVersion := ""
	if options != nil {
		resourceVersion = options.ResourceVersion
		predicate.IncludeUninitialized = options.IncludeUninitialized
	}
	log.Info().Msgf("Watch Predicate: %v", predicate)
	log.Info().Msgf("Watch resource version: %v", resourceVersion)
	return e.WatchPredicate(ctx, predicate, resourceVersion)
}

func (e *REST) Delete(ctx genericapirequest.Context, name string, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	log := logging.DefaultLogger()
	log.Info().Msgf("Inside custom Delete handler")
	key, err := e.KeyFunc(ctx, name)
	log.Info().Msgf("Key: %s", key)

	if err != nil {
		return nil, false, err
	}
	obj := e.NewFunc()
	qualifiedResource := e.qualifiedResourceFromContext(ctx)
	if err := e.Storage.Get(ctx, key, "", obj, false); err != nil {
		return nil, false, storeerr.InterpretDeleteError(err, qualifiedResource, name)
	}
	// support older consumers of delete by treating "nil" as delete immediately
	if options == nil {
		options = metav1.NewDeleteOptions(0)
	}
	var preconditions storage.Preconditions
	if options.Preconditions != nil {
		preconditions.UID = options.Preconditions.UID
	}
	graceful, pendingGraceful, err := rest.BeforeDelete(e.DeleteStrategy, ctx, obj, options)
	if err != nil {
		return nil, false, err
	}
	// this means finalizers cannot be updated via DeleteOptions if a deletion is already pending
	if pendingGraceful {
		out, err := e.finalizeDelete(ctx, obj, false)
		return out, false, err
	}
	// check if obj has pending finalizers
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return nil, false, kubeerr.NewInternalError(err)
	}
	pendingFinalizers := len(accessor.GetFinalizers()) != 0
	var ignoreNotFound bool
	var deleteImmediately bool = true
	var lastExisting, out runtime.Object

	// Handle combinations of graceful deletion and finalization by issuing
	// the correct updates.
	shouldUpdateFinalizers, _ := deletionFinalizersForGarbageCollection(e, accessor, options)
	// TODO: remove the check, because we support no-op updates now.
	if graceful || pendingFinalizers || shouldUpdateFinalizers {
		err, ignoreNotFound, deleteImmediately, out, lastExisting = e.updateForGracefulDeletionAndFinalizers(ctx, name, key, options, preconditions, obj)
	}

	// !deleteImmediately covers all cases where err != nil. We keep both to be future-proof.
	if !deleteImmediately || err != nil {
		return out, false, err
	}

	// delete immediately, or no graceful deletion supported
	glog.V(6).Infof("going to delete %s from registry: ", name)
	out = e.NewFunc()
	if err := e.Storage.Delete(ctx, key, out, &preconditions); err != nil {
		// Please refer to the place where we set ignoreNotFound for the reason
		// why we ignore the NotFound error .
		if storage.IsNotFound(err) && ignoreNotFound && lastExisting != nil {
			// The lastExisting object may not be the last state of the object
			// before its deletion, but it's the best approximation.
			out, err := e.finalizeDelete(ctx, lastExisting, true)
			return out, true, err
		}
		return nil, false, storeerr.InterpretDeleteError(err, qualifiedResource, name)
	}
	out, err = e.finalizeDelete(ctx, out, true)
	return out, true, err
}

func (e *REST) Export(ctx genericapirequest.Context, name string, opts metav1.ExportOptions) (runtime.Object, error) {
	log := logging.DefaultLogger()
	log.Info().Msgf("Inside custom Export handler")
	obj, err := e.Get(ctx, name, &metav1.GetOptions{})
	if err != nil {
		log.Info().Reason(err).Msgf("Error getting object")
		return nil, err
	}
	if accessor, err := meta.Accessor(obj); err == nil {
		exportObjectMeta(accessor, opts.Exact)
	} else {
		glog.V(4).Infof("Object of type %v does not have ObjectMeta: %v", reflect.TypeOf(obj), err)
	}

	if e.ExportStrategy != nil {
		if err = e.ExportStrategy.Export(ctx, obj, opts.Exact); err != nil {
			return nil, err
		}
	} else {
		e.CreateStrategy.PrepareForCreate(ctx, obj)
	}
	log.Info().Msgf("Object: %v", obj)
	return obj, nil
}

// qualifiedResourceFromContext attempts to retrieve a GroupResource from the context's request info.
// If the context has no request info, DefaultQualifiedResource is used.
func (e *REST) qualifiedResourceFromContext(ctx genericapirequest.Context) schema.GroupResource {
	if info, ok := request.RequestInfoFrom(ctx); ok {
		return schema.GroupResource{Group: info.APIGroup, Resource: info.Resource}
	}
	// some implementations access storage directly and thus the context has no RequestInfo
	return e.DefaultQualifiedResource
}

// calculateTTL is a helper for retrieving the updated TTL for an object or
// returning an error if the TTL cannot be calculated. The defaultTTL is
// changed to 1 if less than zero. Zero means no TTL, not expire immediately.
func (e *REST) calculateTTL(obj runtime.Object, defaultTTL int64, update bool) (ttl uint64, err error) {
	// TODO: validate this is assertion is still valid.

	// etcd may return a negative TTL for a node if the expiration has not
	// occurred due to server lag - we will ensure that the value is at least
	// set.
	if defaultTTL < 0 {
		defaultTTL = 1
	}
	ttl = uint64(defaultTTL)
	if e.TTLFunc != nil {
		ttl, err = e.TTLFunc(obj, ttl, update)
	}
	return ttl, err
}

// shouldDeleteDuringUpdate checks if a Update is removing all the object's
// finalizers. If so, it further checks if the object's
// DeletionGracePeriodSeconds is 0.
func (e *REST) shouldDeleteDuringUpdate(ctx genericapirequest.Context, key string, obj, existing runtime.Object) bool {
	newMeta, err := meta.Accessor(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return false
	}
	oldMeta, err := meta.Accessor(existing)
	if err != nil {
		utilruntime.HandleError(err)
		return false
	}
	return len(newMeta.GetFinalizers()) == 0 && oldMeta.GetDeletionGracePeriodSeconds() != nil && *oldMeta.GetDeletionGracePeriodSeconds() == 0
}

// deleteWithoutFinalizers handles deleting an object ignoring its finalizer list.
// Used for objects that are either been finalized or have never initialized.
func (e *REST) deleteWithoutFinalizers(ctx genericapirequest.Context, name, key string, obj runtime.Object, preconditions *storage.Preconditions) (runtime.Object, bool, error) {
	out := e.NewFunc()
	glog.V(6).Infof("going to delete %s from registry, triggered by update", name)
	if err := e.Storage.Delete(ctx, key, out, preconditions); err != nil {
		// Deletion is racy, i.e., there could be multiple update
		// requests to remove all finalizers from the object, so we
		// ignore the NotFound error.
		if storage.IsNotFound(err) {
			_, err := e.finalizeDelete(ctx, obj, true)
			// clients are expecting an updated object if a PUT succeeded,
			// but finalizeDelete returns a metav1.Status, so return
			// the object in the request instead.
			return obj, false, err
		}
		return nil, false, storeerr.InterpretDeleteError(err, e.qualifiedResourceFromContext(ctx), name)
	}
	_, err := e.finalizeDelete(ctx, out, true)
	// clients are expecting an updated object if a PUT succeeded, but
	// finalizeDelete returns a metav1.Status, so return the object in
	// the request instead.
	return obj, false, err
}

// finalizeDelete runs the Store's AfterDelete hook if runHooks is set and
// returns the decorated deleted object if appropriate.
func (e *REST) finalizeDelete(ctx genericapirequest.Context, obj runtime.Object, runHooks bool) (runtime.Object, error) {
	if runHooks && e.AfterDelete != nil {
		if err := e.AfterDelete(obj); err != nil {
			return nil, err
		}
	}
	if e.ReturnDeletedObject {
		if e.Decorator != nil {
			if err := e.Decorator(obj); err != nil {
				return nil, err
			}
		}
		return obj, nil
	}
	// Return information about the deleted object, which enables clients to
	// verify that the object was actually deleted and not waiting for finalizers.
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}
	qualifiedResource := e.qualifiedResourceFromContext(ctx)
	details := &metav1.StatusDetails{
		Name:  accessor.GetName(),
		Group: qualifiedResource.Group,
		Kind:  qualifiedResource.Resource, // Yes we set Kind field to resource.
		UID:   accessor.GetUID(),
	}
	status := &metav1.Status{Status: metav1.StatusSuccess, Details: details}
	return status, nil
}

// shouldDeleteForFailedInitialization returns true if the provided object is initializing and has
// a failure recorded.
func (e *REST) shouldDeleteForFailedInitialization(ctx genericapirequest.Context, obj runtime.Object) bool {
	m, err := meta.Accessor(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return false
	}
	if initializers := m.GetInitializers(); initializers != nil && initializers.Result != nil {
		return true
	}
	return false
}

// WaitForInitialized holds until the object is initialized, or returns an error if the default limit expires.
// This method is exposed publicly for consumers of generic rest tooling.
func (e *REST) WaitForInitialized(ctx genericapirequest.Context, obj runtime.Object) (runtime.Object, error) {
	log := logging.DefaultLogger()
	log.Info().Msgf("Inside custom wait handler")
	// return early if we don't have initializers, or if they've completed already
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return obj, nil
	}

	initializers := accessor.GetInitializers()
	if initializers == nil {
		return obj, nil
	}
	if result := initializers.Result; result != nil {
		return nil, kubeerr.FromObject(result)
	}

	key, err := e.KeyFunc(ctx, accessor.GetName())
	if err != nil {
		return nil, err
	}
	qualifiedResource := e.qualifiedResourceFromContext(ctx)
	w, err := e.Storage.Watch(ctx, key, accessor.GetResourceVersion(), storage.SelectionPredicate{
		Label: labels.Everything(),
		Field: fields.Everything(),

		IncludeUninitialized: true,
	})
	if err != nil {
		return nil, err
	}
	defer w.Stop()

	latest := obj
	ch := w.ResultChan()
	for {
		select {
		case event, ok := <-ch:
			if !ok {
				msg := fmt.Sprintf("server has timed out waiting for the initialization of %s %s",
					qualifiedResource.String(), accessor.GetName())
				return nil, kubeerr.NewTimeoutError(msg, 0)
			}
			switch event.Type {
			case watch.Deleted:
				if latest = event.Object; latest != nil {
					if accessor, err := meta.Accessor(latest); err == nil {
						if initializers := accessor.GetInitializers(); initializers != nil && initializers.Result != nil {
							// initialization failed, but we missed the modification event
							return nil, kubeerr.FromObject(initializers.Result)
						}
					}
				}
				return nil, kubeerr.NewInternalError(fmt.Errorf("object deleted while waiting for creation"))
			case watch.Error:
				if status, ok := event.Object.(*metav1.Status); ok {
					return nil, &kubeerr.StatusError{ErrStatus: *status}
				}
				return nil, kubeerr.NewInternalError(fmt.Errorf("unexpected object in watch stream, can't complete initialization %T", event.Object))
			case watch.Modified:
				latest = event.Object
				accessor, err = meta.Accessor(latest)
				if err != nil {
					return nil, kubeerr.NewInternalError(fmt.Errorf("object no longer has access to metadata %T: %v", latest, err))
				}
				initializers := accessor.GetInitializers()
				if initializers == nil {
					// completed initialization
					return latest, nil
				}
				if result := initializers.Result; result != nil {
					// initialization failed
					return nil, kubeerr.FromObject(result)
				}
			}
		case <-ctx.Done():
		}
	}
}

// objectMetaAndKind retrieves kind and ObjectMeta from a runtime object, or returns an error.
func objectMetaAndKind(typer runtime.ObjectTyper, obj runtime.Object) (metav1.Object, schema.GroupVersionKind, error) {
	objectMeta, err := meta.Accessor(obj)
	if err != nil {
		return nil, schema.GroupVersionKind{}, errors.NewInternalError(err)
	}
	kinds, _, err := typer.ObjectKinds(obj)
	if err != nil {
		return nil, schema.GroupVersionKind{}, errors.NewInternalError(err)
	}
	return objectMeta, kinds[0], nil
}

func deletionFinalizersForGarbageCollection(e *REST, accessor metav1.Object, options *metav1.DeleteOptions) (bool, []string) {
	if !e.EnableGarbageCollection {
		return false, []string{}
	}
	shouldOrphan := shouldOrphanDependents(e, accessor, options)
	shouldDeleteDependentInForeground := shouldDeleteDependents(e, accessor, options)
	newFinalizers := []string{}

	// first remove both finalizers, add them back if needed.
	for _, f := range accessor.GetFinalizers() {
		if f == metav1.FinalizerOrphanDependents || f == metav1.FinalizerDeleteDependents {
			continue
		}
		newFinalizers = append(newFinalizers, f)
	}

	if shouldOrphan {
		newFinalizers = append(newFinalizers, metav1.FinalizerOrphanDependents)
	}
	if shouldDeleteDependentInForeground {
		newFinalizers = append(newFinalizers, metav1.FinalizerDeleteDependents)
	}

	oldFinalizerSet := sets.NewString(accessor.GetFinalizers()...)
	newFinalizersSet := sets.NewString(newFinalizers...)
	if oldFinalizerSet.Equal(newFinalizersSet) {
		return false, accessor.GetFinalizers()
	}
	return true, newFinalizers
}

func shouldOrphanDependents(e *REST, accessor metav1.Object, options *metav1.DeleteOptions) bool {
	if gcStrategy, ok := e.DeleteStrategy.(rest.GarbageCollectionDeleteStrategy); ok {
		if gcStrategy.DefaultGarbageCollectionPolicy() == rest.Unsupported {
			// return  false to indicate that we should NOT orphan
			return false
		}
	}

	// An explicit policy was set at deletion time, that overrides everything
	if options != nil && options.OrphanDependents != nil {
		return *options.OrphanDependents
	}
	if options != nil && options.PropagationPolicy != nil {
		switch *options.PropagationPolicy {
		case metav1.DeletePropagationOrphan:
			return true
		case metav1.DeletePropagationBackground, metav1.DeletePropagationForeground:
			return false
		}
	}

	// If a finalizer is set in the object, it overrides the default
	// validation should make sure the two cases won't be true at the same time.
	finalizers := accessor.GetFinalizers()
	for _, f := range finalizers {
		switch f {
		case metav1.FinalizerOrphanDependents:
			return true
		case metav1.FinalizerDeleteDependents:
			return false
		}
	}

	// Get default orphan policy from this REST object type if it exists
	if gcStrategy, ok := e.DeleteStrategy.(rest.GarbageCollectionDeleteStrategy); ok {
		if gcStrategy.DefaultGarbageCollectionPolicy() == rest.OrphanDependents {
			return true
		}
	}
	return false
}

func shouldDeleteDependents(e *REST, accessor metav1.Object, options *metav1.DeleteOptions) bool {
	// Get default orphan policy from this REST object type
	if gcStrategy, ok := e.DeleteStrategy.(rest.GarbageCollectionDeleteStrategy); ok && gcStrategy.DefaultGarbageCollectionPolicy() == rest.Unsupported {
		// return false to indicate that we should NOT delete in foreground
		return false
	}

	// If an explicit policy was set at deletion time, that overrides both
	if options != nil && options.OrphanDependents != nil {
		return false
	}
	if options != nil && options.PropagationPolicy != nil {
		switch *options.PropagationPolicy {
		case metav1.DeletePropagationForeground:
			return true
		case metav1.DeletePropagationBackground, metav1.DeletePropagationOrphan:
			return false
		}
	}

	// If a finalizer is set in the object, it overrides the default
	// validation has made sure the two cases won't be true at the same time.
	finalizers := accessor.GetFinalizers()
	for _, f := range finalizers {
		switch f {
		case metav1.FinalizerDeleteDependents:
			return true
		case metav1.FinalizerOrphanDependents:
			return false
		}
	}

	return false
}

func (e *REST) updateForGracefulDeletionAndFinalizers(ctx genericapirequest.Context, name, key string, options *metav1.DeleteOptions, preconditions storage.Preconditions, in runtime.Object) (err error, ignoreNotFound, deleteImmediately bool, out, lastExisting runtime.Object) {
	lastGraceful := int64(0)
	var pendingFinalizers bool
	out = e.NewFunc()
	err = e.Storage.GuaranteedUpdate(
		ctx,
		key,
		out,
		false, // ignoreNotFound
		&preconditions,
		storage.SimpleUpdate(func(existing runtime.Object) (runtime.Object, error) {
			graceful, pendingGraceful, err := rest.BeforeDelete(e.DeleteStrategy, ctx, existing, options)
			if err != nil {
				return nil, err
			}
			if pendingGraceful {
				return nil, errAlreadyDeleting
			}

			// Add/remove the orphan finalizer as the options dictates.
			// Note that this occurs after checking pendingGraceufl, so
			// finalizers cannot be updated via DeleteOptions if deletion has
			// started.
			existingAccessor, err := meta.Accessor(existing)
			if err != nil {
				return nil, err
			}
			needsUpdate, newFinalizers := deletionFinalizersForGarbageCollection(e, existingAccessor, options)
			if needsUpdate {
				existingAccessor.SetFinalizers(newFinalizers)
			}

			pendingFinalizers = len(existingAccessor.GetFinalizers()) != 0
			if !graceful {
				// set the DeleteGracePeriods to 0 if the object has pendingFinalizers but not supporting graceful deletion
				if pendingFinalizers {
					glog.V(6).Infof("update the DeletionTimestamp to \"now\" and GracePeriodSeconds to 0 for object %s, because it has pending finalizers", name)
					err = markAsDeleting(existing)
					if err != nil {
						return nil, err
					}
					return existing, nil
				}
				return nil, errDeleteNow
			}
			lastGraceful = *options.GracePeriodSeconds
			lastExisting = existing
			return existing, nil
		}),
	)
	switch err {
	case nil:
		// If there are pending finalizers, we never delete the object immediately.
		if pendingFinalizers {
			return nil, false, false, out, lastExisting
		}
		if lastGraceful > 0 {
			return nil, false, false, out, lastExisting
		}
		// If we are here, the registry supports grace period mechanism and
		// we are intentionally delete gracelessly. In this case, we may
		// enter a race with other k8s components. If other component wins
		// the race, the object will not be found, and we should tolerate
		// the NotFound error. See
		// https://github.com/kubernetes/kubernetes/issues/19403 for
		// details.
		return nil, true, true, out, lastExisting
	case errDeleteNow:
		// we've updated the object to have a zero grace period, or it's already at 0, so
		// we should fall through and truly delete the object.
		return nil, false, true, out, lastExisting
	case errAlreadyDeleting:
		out, err = e.finalizeDelete(ctx, in, true)
		return err, false, false, out, lastExisting
	default:
		return storeerr.InterpretUpdateError(err, e.qualifiedResourceFromContext(ctx), name), false, false, out, lastExisting
	}
}

func markAsDeleting(obj runtime.Object) (err error) {
	objectMeta, kerr := meta.Accessor(obj)
	if kerr != nil {
		return kerr
	}
	now := metav1.NewTime(time.Now())
	// This handles Generation bump for resources that don't support graceful
	// deletion. For resources that support graceful deletion is handle in
	// pkg/api/rest/delete.go
	if objectMeta.GetDeletionTimestamp() == nil && objectMeta.GetGeneration() > 0 {
		objectMeta.SetGeneration(objectMeta.GetGeneration() + 1)
	}
	objectMeta.SetDeletionTimestamp(&now)
	var zero int64 = 0
	objectMeta.SetDeletionGracePeriodSeconds(&zero)
	return nil
}

func exportObjectMeta(accessor metav1.Object, exact bool) {
	accessor.SetUID("")
	if !exact {
		accessor.SetNamespace("")
	}
	accessor.SetCreationTimestamp(metav1.Time{})
	accessor.SetDeletionTimestamp(nil)
	accessor.SetResourceVersion("")
	accessor.SetSelfLink("")
	if len(accessor.GetGenerateName()) > 0 && !exact {
		accessor.SetName("")
	}
}
