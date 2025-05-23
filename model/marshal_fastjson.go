// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Code generated by "generate-fastjson". DO NOT EDIT.

package model

import (
	"errors"
	"math"

	"go.elastic.co/fastjson"
)

var (
	_ = errors.New
	_ = math.IsNaN
)

func (v *Service) MarshalFastJSON(w *fastjson.Writer) error {
	var firstErr error
	w.RawByte('{')
	first := true
	if v.Agent != nil {
		const prefix = ",\"agent\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		if err := v.Agent.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Environment != "" {
		const prefix = ",\"environment\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Environment)
	}
	if v.Framework != nil {
		const prefix = ",\"framework\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		if err := v.Framework.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Language != nil {
		const prefix = ",\"language\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		if err := v.Language.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Name != "" {
		const prefix = ",\"name\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Name)
	}
	if v.Node != nil {
		const prefix = ",\"node\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		if err := v.Node.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Runtime != nil {
		const prefix = ",\"runtime\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		if err := v.Runtime.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Version != "" {
		const prefix = ",\"version\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Version)
	}
	w.RawByte('}')
	return firstErr
}

func (v *Agent) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	w.RawString("\"name\":")
	w.String(v.Name)
	w.RawString(",\"version\":")
	w.String(v.Version)
	w.RawByte('}')
	return nil
}

func (v *Framework) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	w.RawString("\"name\":")
	w.String(v.Name)
	w.RawString(",\"version\":")
	w.String(v.Version)
	w.RawByte('}')
	return nil
}

func (v *Language) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	w.RawString("\"name\":")
	w.String(v.Name)
	if v.Version != "" {
		w.RawString(",\"version\":")
		w.String(v.Version)
	}
	w.RawByte('}')
	return nil
}

func (v *Runtime) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	w.RawString("\"name\":")
	w.String(v.Name)
	w.RawString(",\"version\":")
	w.String(v.Version)
	w.RawByte('}')
	return nil
}

func (v *ServiceNode) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	if v.ConfiguredName != "" {
		w.RawString("\"configured_name\":")
		w.String(v.ConfiguredName)
	}
	w.RawByte('}')
	return nil
}

func (v *System) MarshalFastJSON(w *fastjson.Writer) error {
	var firstErr error
	w.RawByte('{')
	first := true
	if v.Architecture != "" {
		const prefix = ",\"architecture\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Architecture)
	}
	if v.Container != nil {
		const prefix = ",\"container\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		if err := v.Container.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Hostname != "" {
		const prefix = ",\"hostname\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Hostname)
	}
	if v.Kubernetes != nil {
		const prefix = ",\"kubernetes\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		if err := v.Kubernetes.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Platform != "" {
		const prefix = ",\"platform\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Platform)
	}
	w.RawByte('}')
	return firstErr
}

func (v *Process) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	w.RawString("\"pid\":")
	w.Int64(int64(v.Pid))
	if v.Argv != nil {
		w.RawString(",\"argv\":")
		w.RawByte('[')
		for i, v := range v.Argv {
			if i != 0 {
				w.RawByte(',')
			}
			w.String(v)
		}
		w.RawByte(']')
	}
	if v.Ppid != nil {
		w.RawString(",\"ppid\":")
		w.Int64(int64(*v.Ppid))
	}
	if v.Title != "" {
		w.RawString(",\"title\":")
		w.String(v.Title)
	}
	w.RawByte('}')
	return nil
}

func (v *Container) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	w.RawString("\"id\":")
	w.String(v.ID)
	w.RawByte('}')
	return nil
}

func (v *Kubernetes) MarshalFastJSON(w *fastjson.Writer) error {
	var firstErr error
	w.RawByte('{')
	first := true
	if v.Namespace != "" {
		const prefix = ",\"namespace\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Namespace)
	}
	if v.Node != nil {
		const prefix = ",\"node\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		if err := v.Node.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Pod != nil {
		const prefix = ",\"pod\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		if err := v.Pod.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	w.RawByte('}')
	return firstErr
}

func (v *KubernetesNode) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	if v.Name != "" {
		w.RawString("\"name\":")
		w.String(v.Name)
	}
	w.RawByte('}')
	return nil
}

func (v *KubernetesPod) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	first := true
	if v.Name != "" {
		const prefix = ",\"name\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Name)
	}
	if v.UID != "" {
		const prefix = ",\"uid\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.UID)
	}
	w.RawByte('}')
	return nil
}

func (v *Cloud) MarshalFastJSON(w *fastjson.Writer) error {
	var firstErr error
	w.RawByte('{')
	w.RawString("\"provider\":")
	w.String(v.Provider)
	if v.Account != nil {
		w.RawString(",\"account\":")
		if err := v.Account.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.AvailabilityZone != "" {
		w.RawString(",\"availability_zone\":")
		w.String(v.AvailabilityZone)
	}
	if v.Instance != nil {
		w.RawString(",\"instance\":")
		if err := v.Instance.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Machine != nil {
		w.RawString(",\"machine\":")
		if err := v.Machine.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Project != nil {
		w.RawString(",\"project\":")
		if err := v.Project.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Region != "" {
		w.RawString(",\"region\":")
		w.String(v.Region)
	}
	w.RawByte('}')
	return firstErr
}

func (v *CloudInstance) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	first := true
	if v.ID != "" {
		const prefix = ",\"id\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.ID)
	}
	if v.Name != "" {
		const prefix = ",\"name\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Name)
	}
	w.RawByte('}')
	return nil
}

func (v *CloudMachine) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	if v.Type != "" {
		w.RawString("\"type\":")
		w.String(v.Type)
	}
	w.RawByte('}')
	return nil
}

func (v *CloudAccount) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	first := true
	if v.ID != "" {
		const prefix = ",\"id\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.ID)
	}
	if v.Name != "" {
		const prefix = ",\"name\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Name)
	}
	w.RawByte('}')
	return nil
}

func (v *CloudProject) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	first := true
	if v.ID != "" {
		const prefix = ",\"id\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.ID)
	}
	if v.Name != "" {
		const prefix = ",\"name\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Name)
	}
	w.RawByte('}')
	return nil
}

func (v *Transaction) MarshalFastJSON(w *fastjson.Writer) error {
	var firstErr error
	w.RawByte('{')
	w.RawString("\"duration\":")
	if math.IsNaN(v.Duration) {
		return errors.New("json: 'v.Duration': unsupported value: NaN")
	}
	if math.IsInf(v.Duration, 0) {
		return errors.New("json: 'v.Duration': unsupported value: Inf")
	}
	w.Float64(v.Duration)
	w.RawString(",\"id\":")
	if err := v.ID.MarshalFastJSON(w); err != nil && firstErr == nil {
		firstErr = err
	}
	w.RawString(",\"name\":")
	w.String(v.Name)
	w.RawString(",\"span_count\":")
	if err := v.SpanCount.MarshalFastJSON(w); err != nil && firstErr == nil {
		firstErr = err
	}
	w.RawString(",\"timestamp\":")
	if err := v.Timestamp.MarshalFastJSON(w); err != nil && firstErr == nil {
		firstErr = err
	}
	w.RawString(",\"trace_id\":")
	if err := v.TraceID.MarshalFastJSON(w); err != nil && firstErr == nil {
		firstErr = err
	}
	w.RawString(",\"type\":")
	w.String(v.Type)
	if v.Context != nil {
		w.RawString(",\"context\":")
		if err := v.Context.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.DroppedSpansStats != nil {
		w.RawString(",\"dropped_spans_stats\":")
		w.RawByte('[')
		for i, v := range v.DroppedSpansStats {
			if i != 0 {
				w.RawByte(',')
			}
			if err := v.MarshalFastJSON(w); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		w.RawByte(']')
	}
	if v.FAAS != nil {
		w.RawString(",\"faas\":")
		if err := v.FAAS.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Links != nil {
		w.RawString(",\"links\":")
		w.RawByte('[')
		for i, v := range v.Links {
			if i != 0 {
				w.RawByte(',')
			}
			if err := v.MarshalFastJSON(w); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		w.RawByte(']')
	}
	if v.OTel != nil {
		w.RawString(",\"otel\":")
		if err := v.OTel.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Outcome != "" {
		w.RawString(",\"outcome\":")
		w.String(v.Outcome)
	}
	if !v.ParentID.isZero() {
		w.RawString(",\"parent_id\":")
		if err := v.ParentID.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Result != "" {
		w.RawString(",\"result\":")
		w.String(v.Result)
	}
	if v.SampleRate != nil {
		w.RawString(",\"sample_rate\":")
		if math.IsNaN(*v.SampleRate) {
			return errors.New("json: '*v.SampleRate': unsupported value: NaN")
		}
		if math.IsInf(*v.SampleRate, 0) {
			return errors.New("json: '*v.SampleRate': unsupported value: Inf")
		}
		w.Float64(*v.SampleRate)
	}
	if v.Sampled != nil {
		w.RawString(",\"sampled\":")
		w.Bool(*v.Sampled)
	}
	w.RawByte('}')
	return firstErr
}

func (v *OTel) MarshalFastJSON(w *fastjson.Writer) error {
	var firstErr error
	w.RawByte('{')
	w.RawString("\"span_kind\":")
	w.String(v.SpanKind)
	if v.Attributes != nil {
		w.RawString(",\"attributes\":")
		w.RawByte('{')
		{
			first := true
			for k, v := range v.Attributes {
				if first {
					first = false
				} else {
					w.RawByte(',')
				}
				w.String(k)
				w.RawByte(':')
				if err := fastjson.Marshal(w, v); err != nil && firstErr == nil {
					firstErr = err
				}
			}
		}
		w.RawByte('}')
	}
	w.RawByte('}')
	return firstErr
}

func (v *SpanCount) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	w.RawString("\"dropped\":")
	w.Int64(int64(v.Dropped))
	w.RawString(",\"started\":")
	w.Int64(int64(v.Started))
	w.RawByte('}')
	return nil
}

func (v *DroppedSpansStats) MarshalFastJSON(w *fastjson.Writer) error {
	var firstErr error
	w.RawByte('{')
	w.RawString("\"destination_service_resource\":")
	w.String(v.DestinationServiceResource)
	w.RawString(",\"duration\":")
	if err := v.Duration.MarshalFastJSON(w); err != nil && firstErr == nil {
		firstErr = err
	}
	w.RawString(",\"outcome\":")
	w.String(v.Outcome)
	if v.ServiceTargetName != "" {
		w.RawString(",\"service_target_name\":")
		w.String(v.ServiceTargetName)
	}
	if v.ServiceTargetType != "" {
		w.RawString(",\"service_target_type\":")
		w.String(v.ServiceTargetType)
	}
	w.RawByte('}')
	return firstErr
}

func (v *AggregateDuration) MarshalFastJSON(w *fastjson.Writer) error {
	var firstErr error
	w.RawByte('{')
	w.RawString("\"count\":")
	w.Int64(int64(v.Count))
	w.RawString(",\"sum\":")
	if err := v.Sum.MarshalFastJSON(w); err != nil && firstErr == nil {
		firstErr = err
	}
	w.RawByte('}')
	return firstErr
}

func (v *DurationSum) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	w.RawString("\"us\":")
	w.Int64(v.Us)
	w.RawByte('}')
	return nil
}

func (v *Span) MarshalFastJSON(w *fastjson.Writer) error {
	var firstErr error
	w.RawByte('{')
	w.RawString("\"duration\":")
	if math.IsNaN(v.Duration) {
		return errors.New("json: 'v.Duration': unsupported value: NaN")
	}
	if math.IsInf(v.Duration, 0) {
		return errors.New("json: 'v.Duration': unsupported value: Inf")
	}
	w.Float64(v.Duration)
	w.RawString(",\"id\":")
	if err := v.ID.MarshalFastJSON(w); err != nil && firstErr == nil {
		firstErr = err
	}
	w.RawString(",\"name\":")
	w.String(v.Name)
	w.RawString(",\"timestamp\":")
	if err := v.Timestamp.MarshalFastJSON(w); err != nil && firstErr == nil {
		firstErr = err
	}
	w.RawString(",\"trace_id\":")
	if err := v.TraceID.MarshalFastJSON(w); err != nil && firstErr == nil {
		firstErr = err
	}
	w.RawString(",\"type\":")
	w.String(v.Type)
	if v.Action != "" {
		w.RawString(",\"action\":")
		w.String(v.Action)
	}
	if v.Composite != nil {
		w.RawString(",\"composite\":")
		if err := v.Composite.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Context != nil {
		w.RawString(",\"context\":")
		if err := v.Context.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Links != nil {
		w.RawString(",\"links\":")
		w.RawByte('[')
		for i, v := range v.Links {
			if i != 0 {
				w.RawByte(',')
			}
			if err := v.MarshalFastJSON(w); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		w.RawByte(']')
	}
	if v.OTel != nil {
		w.RawString(",\"otel\":")
		if err := v.OTel.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Outcome != "" {
		w.RawString(",\"outcome\":")
		w.String(v.Outcome)
	}
	if !v.ParentID.isZero() {
		w.RawString(",\"parent_id\":")
		if err := v.ParentID.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.SampleRate != nil {
		w.RawString(",\"sample_rate\":")
		if math.IsNaN(*v.SampleRate) {
			return errors.New("json: '*v.SampleRate': unsupported value: NaN")
		}
		if math.IsInf(*v.SampleRate, 0) {
			return errors.New("json: '*v.SampleRate': unsupported value: Inf")
		}
		w.Float64(*v.SampleRate)
	}
	if v.Stacktrace != nil {
		w.RawString(",\"stacktrace\":")
		w.RawByte('[')
		for i, v := range v.Stacktrace {
			if i != 0 {
				w.RawByte(',')
			}
			if err := v.MarshalFastJSON(w); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		w.RawByte(']')
	}
	if v.Subtype != "" {
		w.RawString(",\"subtype\":")
		w.String(v.Subtype)
	}
	if !v.TransactionID.isZero() {
		w.RawString(",\"transaction_id\":")
		if err := v.TransactionID.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	w.RawByte('}')
	return firstErr
}

func (v *SpanContext) MarshalFastJSON(w *fastjson.Writer) error {
	var firstErr error
	w.RawByte('{')
	first := true
	if v.Database != nil {
		const prefix = ",\"db\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		if err := v.Database.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Destination != nil {
		const prefix = ",\"destination\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		if err := v.Destination.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.HTTP != nil {
		const prefix = ",\"http\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		if err := v.HTTP.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Message != nil {
		const prefix = ",\"message\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		if err := v.Message.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Service != nil {
		const prefix = ",\"service\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		if err := v.Service.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if !v.Tags.isZero() {
		const prefix = ",\"tags\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		if err := v.Tags.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	w.RawByte('}')
	return firstErr
}

func (v *SpanLink) MarshalFastJSON(w *fastjson.Writer) error {
	var firstErr error
	w.RawByte('{')
	w.RawString("\"span_id\":")
	if err := v.SpanID.MarshalFastJSON(w); err != nil && firstErr == nil {
		firstErr = err
	}
	w.RawString(",\"trace_id\":")
	if err := v.TraceID.MarshalFastJSON(w); err != nil && firstErr == nil {
		firstErr = err
	}
	w.RawByte('}')
	return firstErr
}

func (v *DestinationSpanContext) MarshalFastJSON(w *fastjson.Writer) error {
	var firstErr error
	w.RawByte('{')
	first := true
	if v.Address != "" {
		const prefix = ",\"address\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Address)
	}
	if v.Cloud != nil {
		const prefix = ",\"cloud\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		if err := v.Cloud.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Port != 0 {
		const prefix = ",\"port\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.Int64(int64(v.Port))
	}
	if v.Service != nil {
		const prefix = ",\"service\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		if err := v.Service.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	w.RawByte('}')
	return firstErr
}

func (v *ServiceSpanContext) MarshalFastJSON(w *fastjson.Writer) error {
	var firstErr error
	w.RawByte('{')
	if v.Target != nil {
		w.RawString("\"target\":")
		if err := v.Target.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	w.RawByte('}')
	return firstErr
}

func (v *ServiceTargetSpanContext) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	w.RawString("\"type\":")
	w.String(v.Type)
	if v.Name != "" {
		w.RawString(",\"name\":")
		w.String(v.Name)
	}
	w.RawByte('}')
	return nil
}

func (v *DestinationServiceSpanContext) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	w.RawString("\"name\":")
	w.String(v.Name)
	if v.Resource != "" {
		w.RawString(",\"resource\":")
		w.String(v.Resource)
	}
	if v.Type != "" {
		w.RawString(",\"type\":")
		w.String(v.Type)
	}
	w.RawByte('}')
	return nil
}

func (v *DestinationCloudSpanContext) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	if v.Region != "" {
		w.RawString("\"region\":")
		w.String(v.Region)
	}
	w.RawByte('}')
	return nil
}

func (v *MessageSpanContext) MarshalFastJSON(w *fastjson.Writer) error {
	var firstErr error
	w.RawByte('{')
	if v.Queue != nil {
		w.RawString("\"queue\":")
		if err := v.Queue.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	w.RawByte('}')
	return firstErr
}

func (v *MessageQueueSpanContext) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	if v.Name != "" {
		w.RawString("\"name\":")
		w.String(v.Name)
	}
	w.RawByte('}')
	return nil
}

func (v *DatabaseSpanContext) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	first := true
	if v.Instance != "" {
		const prefix = ",\"instance\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Instance)
	}
	if v.RowsAffected != nil {
		const prefix = ",\"rows_affected\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.Int64(*v.RowsAffected)
	}
	if v.Statement != "" {
		const prefix = ",\"statement\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Statement)
	}
	if v.Type != "" {
		const prefix = ",\"type\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Type)
	}
	if v.User != "" {
		const prefix = ",\"user\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.User)
	}
	w.RawByte('}')
	return nil
}

func (v *CompositeSpan) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	w.RawString("\"compression_strategy\":")
	w.String(v.CompressionStrategy)
	w.RawString(",\"count\":")
	w.Int64(int64(v.Count))
	w.RawString(",\"sum\":")
	if math.IsNaN(v.Sum) {
		return errors.New("json: 'v.Sum': unsupported value: NaN")
	}
	if math.IsInf(v.Sum, 0) {
		return errors.New("json: 'v.Sum': unsupported value: Inf")
	}
	w.Float64(v.Sum)
	w.RawByte('}')
	return nil
}

func (v *Context) MarshalFastJSON(w *fastjson.Writer) error {
	var firstErr error
	w.RawByte('{')
	first := true
	if !v.Custom.isZero() {
		const prefix = ",\"custom\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		if err := v.Custom.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Request != nil {
		const prefix = ",\"request\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		if err := v.Request.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Response != nil {
		const prefix = ",\"response\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		if err := v.Response.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Service != nil {
		const prefix = ",\"service\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		if err := v.Service.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if !v.Tags.isZero() {
		const prefix = ",\"tags\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		if err := v.Tags.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.User != nil {
		const prefix = ",\"user\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		if err := v.User.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	w.RawByte('}')
	return firstErr
}

func (v *User) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	first := true
	if v.Email != "" {
		const prefix = ",\"email\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Email)
	}
	if v.ID != "" {
		const prefix = ",\"id\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.ID)
	}
	if v.Username != "" {
		const prefix = ",\"username\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Username)
	}
	w.RawByte('}')
	return nil
}

func (v *Error) MarshalFastJSON(w *fastjson.Writer) error {
	var firstErr error
	w.RawByte('{')
	w.RawString("\"id\":")
	if err := v.ID.MarshalFastJSON(w); err != nil && firstErr == nil {
		firstErr = err
	}
	w.RawString(",\"timestamp\":")
	if err := v.Timestamp.MarshalFastJSON(w); err != nil && firstErr == nil {
		firstErr = err
	}
	if v.Context != nil {
		w.RawString(",\"context\":")
		if err := v.Context.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Culprit != "" {
		w.RawString(",\"culprit\":")
		w.String(v.Culprit)
	}
	if !v.Exception.isZero() {
		w.RawString(",\"exception\":")
		if err := v.Exception.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if !v.Log.isZero() {
		w.RawString(",\"log\":")
		if err := v.Log.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if !v.ParentID.isZero() {
		w.RawString(",\"parent_id\":")
		if err := v.ParentID.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if !v.TraceID.isZero() {
		w.RawString(",\"trace_id\":")
		if err := v.TraceID.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if !v.Transaction.isZero() {
		w.RawString(",\"transaction\":")
		if err := v.Transaction.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if !v.TransactionID.isZero() {
		w.RawString(",\"transaction_id\":")
		if err := v.TransactionID.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	w.RawByte('}')
	return firstErr
}

func (v *ErrorTransaction) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	first := true
	if v.Name != "" {
		const prefix = ",\"name\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Name)
	}
	if v.Sampled != nil {
		const prefix = ",\"sampled\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.Bool(*v.Sampled)
	}
	if v.Type != "" {
		const prefix = ",\"type\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Type)
	}
	w.RawByte('}')
	return nil
}

func (v *Exception) MarshalFastJSON(w *fastjson.Writer) error {
	var firstErr error
	w.RawByte('{')
	w.RawString("\"handled\":")
	w.Bool(v.Handled)
	w.RawString(",\"message\":")
	w.String(v.Message)
	if v.Attributes != nil {
		w.RawString(",\"attributes\":")
		w.RawByte('{')
		{
			first := true
			for k, v := range v.Attributes {
				if first {
					first = false
				} else {
					w.RawByte(',')
				}
				w.String(k)
				w.RawByte(':')
				if err := fastjson.Marshal(w, v); err != nil && firstErr == nil {
					firstErr = err
				}
			}
		}
		w.RawByte('}')
	}
	if v.Cause != nil {
		w.RawString(",\"cause\":")
		w.RawByte('[')
		for i, v := range v.Cause {
			if i != 0 {
				w.RawByte(',')
			}
			if err := v.MarshalFastJSON(w); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		w.RawByte(']')
	}
	if !v.Code.isZero() {
		w.RawString(",\"code\":")
		if err := v.Code.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Module != "" {
		w.RawString(",\"module\":")
		w.String(v.Module)
	}
	if v.Stacktrace != nil {
		w.RawString(",\"stacktrace\":")
		w.RawByte('[')
		for i, v := range v.Stacktrace {
			if i != 0 {
				w.RawByte(',')
			}
			if err := v.MarshalFastJSON(w); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		w.RawByte(']')
	}
	if v.Type != "" {
		w.RawString(",\"type\":")
		w.String(v.Type)
	}
	w.RawByte('}')
	return firstErr
}

func (v *StacktraceFrame) MarshalFastJSON(w *fastjson.Writer) error {
	var firstErr error
	w.RawByte('{')
	w.RawString("\"filename\":")
	w.String(v.File)
	w.RawString(",\"lineno\":")
	w.Int64(int64(v.Line))
	if v.AbsolutePath != "" {
		w.RawString(",\"abs_path\":")
		w.String(v.AbsolutePath)
	}
	if v.Classname != "" {
		w.RawString(",\"classname\":")
		w.String(v.Classname)
	}
	if v.Column != nil {
		w.RawString(",\"colno\":")
		w.Int64(int64(*v.Column))
	}
	if v.ContextLine != "" {
		w.RawString(",\"context_line\":")
		w.String(v.ContextLine)
	}
	if v.Function != "" {
		w.RawString(",\"function\":")
		w.String(v.Function)
	}
	if v.LibraryFrame != false {
		w.RawString(",\"library_frame\":")
		w.Bool(v.LibraryFrame)
	}
	if v.Module != "" {
		w.RawString(",\"module\":")
		w.String(v.Module)
	}
	if v.PostContext != nil {
		w.RawString(",\"post_context\":")
		w.RawByte('[')
		for i, v := range v.PostContext {
			if i != 0 {
				w.RawByte(',')
			}
			w.String(v)
		}
		w.RawByte(']')
	}
	if v.PreContext != nil {
		w.RawString(",\"pre_context\":")
		w.RawByte('[')
		for i, v := range v.PreContext {
			if i != 0 {
				w.RawByte(',')
			}
			w.String(v)
		}
		w.RawByte(']')
	}
	if v.Vars != nil {
		w.RawString(",\"vars\":")
		w.RawByte('{')
		{
			first := true
			for k, v := range v.Vars {
				if first {
					first = false
				} else {
					w.RawByte(',')
				}
				w.String(k)
				w.RawByte(':')
				if err := fastjson.Marshal(w, v); err != nil && firstErr == nil {
					firstErr = err
				}
			}
		}
		w.RawByte('}')
	}
	w.RawByte('}')
	return firstErr
}

func (v *Log) MarshalFastJSON(w *fastjson.Writer) error {
	var firstErr error
	w.RawByte('{')
	w.RawString("\"message\":")
	w.String(v.Message)
	if v.Level != "" {
		w.RawString(",\"level\":")
		w.String(v.Level)
	}
	if v.LoggerName != "" {
		w.RawString(",\"logger_name\":")
		w.String(v.LoggerName)
	}
	if v.ParamMessage != "" {
		w.RawString(",\"param_message\":")
		w.String(v.ParamMessage)
	}
	if v.Stacktrace != nil {
		w.RawString(",\"stacktrace\":")
		w.RawByte('[')
		for i, v := range v.Stacktrace {
			if i != 0 {
				w.RawByte(',')
			}
			if err := v.MarshalFastJSON(w); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		w.RawByte(']')
	}
	w.RawByte('}')
	return firstErr
}

func (v *Request) MarshalFastJSON(w *fastjson.Writer) error {
	var firstErr error
	w.RawByte('{')
	w.RawString("\"method\":")
	w.String(v.Method)
	w.RawString(",\"url\":")
	if err := v.URL.MarshalFastJSON(w); err != nil && firstErr == nil {
		firstErr = err
	}
	if v.Body != nil {
		w.RawString(",\"body\":")
		if err := v.Body.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if !v.Cookies.isZero() {
		w.RawString(",\"cookies\":")
		if err := v.Cookies.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Env != nil {
		w.RawString(",\"env\":")
		w.RawByte('{')
		{
			first := true
			for k, v := range v.Env {
				if first {
					first = false
				} else {
					w.RawByte(',')
				}
				w.String(k)
				w.RawByte(':')
				w.String(v)
			}
		}
		w.RawByte('}')
	}
	if !v.Headers.isZero() {
		w.RawString(",\"headers\":")
		if err := v.Headers.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.HTTPVersion != "" {
		w.RawString(",\"http_version\":")
		w.String(v.HTTPVersion)
	}
	if v.Socket != nil {
		w.RawString(",\"socket\":")
		if err := v.Socket.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	w.RawByte('}')
	return firstErr
}

func (v *RequestSocket) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	first := true
	if v.Encrypted != false {
		const prefix = ",\"encrypted\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.Bool(v.Encrypted)
	}
	if v.RemoteAddress != "" {
		const prefix = ",\"remote_address\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.RemoteAddress)
	}
	w.RawByte('}')
	return nil
}

func (v *Response) MarshalFastJSON(w *fastjson.Writer) error {
	var firstErr error
	w.RawByte('{')
	first := true
	if v.Finished != nil {
		const prefix = ",\"finished\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.Bool(*v.Finished)
	}
	if !v.Headers.isZero() {
		const prefix = ",\"headers\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		if err := v.Headers.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.HeadersSent != nil {
		const prefix = ",\"headers_sent\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.Bool(*v.HeadersSent)
	}
	if v.StatusCode != 0 {
		const prefix = ",\"status_code\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.Int64(int64(v.StatusCode))
	}
	w.RawByte('}')
	return firstErr
}

func (v *Metrics) MarshalFastJSON(w *fastjson.Writer) error {
	var firstErr error
	w.RawByte('{')
	w.RawString("\"samples\":")
	if v.Samples == nil {
		w.RawString("null")
	} else {
		w.RawByte('{')
		{
			first := true
			for k, v := range v.Samples {
				if first {
					first = false
				} else {
					w.RawByte(',')
				}
				w.String(k)
				w.RawByte(':')
				if err := v.MarshalFastJSON(w); err != nil && firstErr == nil {
					firstErr = err
				}
			}
		}
		w.RawByte('}')
	}
	w.RawString(",\"timestamp\":")
	if err := v.Timestamp.MarshalFastJSON(w); err != nil && firstErr == nil {
		firstErr = err
	}
	if v.FAAS != nil {
		w.RawString(",\"faas\":")
		if err := v.FAAS.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if !v.Span.isZero() {
		w.RawString(",\"span\":")
		if err := v.Span.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if !v.Labels.isZero() {
		w.RawString(",\"tags\":")
		if err := v.Labels.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if !v.Transaction.isZero() {
		w.RawString(",\"transaction\":")
		if err := v.Transaction.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	w.RawByte('}')
	return firstErr
}

func (v *MetricsTransaction) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	first := true
	if v.Name != "" {
		const prefix = ",\"name\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Name)
	}
	if v.Type != "" {
		const prefix = ",\"type\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Type)
	}
	w.RawByte('}')
	return nil
}

func (v *MetricsSpan) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	first := true
	if v.Subtype != "" {
		const prefix = ",\"subtype\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Subtype)
	}
	if v.Type != "" {
		const prefix = ",\"type\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Type)
	}
	w.RawByte('}')
	return nil
}

func (v *FAAS) MarshalFastJSON(w *fastjson.Writer) error {
	var firstErr error
	w.RawByte('{')
	w.RawString("\"coldstart\":")
	w.Bool(v.Coldstart)
	if v.Execution != "" {
		w.RawString(",\"execution\":")
		w.String(v.Execution)
	}
	if v.ID != "" {
		w.RawString(",\"id\":")
		w.String(v.ID)
	}
	if v.Name != "" {
		w.RawString(",\"name\":")
		w.String(v.Name)
	}
	if v.Trigger != nil {
		w.RawString(",\"trigger\":")
		if err := v.Trigger.MarshalFastJSON(w); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if v.Version != "" {
		w.RawString(",\"version\":")
		w.String(v.Version)
	}
	w.RawByte('}')
	return firstErr
}

func (v *FAASTrigger) MarshalFastJSON(w *fastjson.Writer) error {
	w.RawByte('{')
	first := true
	if v.RequestID != "" {
		const prefix = ",\"request_id\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.RequestID)
	}
	if v.Type != "" {
		const prefix = ",\"type\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Type)
	}
	w.RawByte('}')
	return nil
}
