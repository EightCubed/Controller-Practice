package v1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

func (in *LogCleaner) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(LogCleaner)
	in.DeepCopyInto(out)
	return out
}

func (in *LogCleaner) DeepCopyInto(out *LogCleaner) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
}

func (in *LogCleanerList) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(LogCleanerList)
	in.DeepCopyInto(out)
	return out
}

func (in *LogCleanerList) DeepCopyInto(out *LogCleanerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]LogCleaner, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}
