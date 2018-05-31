// Code generated by "go generate". DO NOT EDIT.

package model

import (
	"github.com/elastic/apm-agent-go/internal/fastjson"
)

func (v *Service) MarshalFastJSON(w *fastjson.Writer) {
	w.RawByte('{')
	w.RawString("\"agent\":")
	v.Agent.MarshalFastJSON(w)
	w.RawString(",\"name\":")
	w.String(v.Name)
	if v.Environment != "" {
		w.RawString(",\"environment\":")
		w.String(v.Environment)
	}
	if v.Framework != nil {
		w.RawString(",\"framework\":")
		v.Framework.MarshalFastJSON(w)
	}
	if v.Language != nil {
		w.RawString(",\"language\":")
		v.Language.MarshalFastJSON(w)
	}
	if v.Runtime != nil {
		w.RawString(",\"runtime\":")
		v.Runtime.MarshalFastJSON(w)
	}
	if v.Version != "" {
		w.RawString(",\"version\":")
		w.String(v.Version)
	}
	w.RawByte('}')
}

func (v *Agent) MarshalFastJSON(w *fastjson.Writer) {
	w.RawByte('{')
	w.RawString("\"name\":")
	w.String(v.Name)
	w.RawString(",\"version\":")
	w.String(v.Version)
	w.RawByte('}')
}

func (v *Framework) MarshalFastJSON(w *fastjson.Writer) {
	w.RawByte('{')
	w.RawString("\"name\":")
	w.String(v.Name)
	w.RawString(",\"version\":")
	w.String(v.Version)
	w.RawByte('}')
}

func (v *Language) MarshalFastJSON(w *fastjson.Writer) {
	w.RawByte('{')
	w.RawString("\"name\":")
	w.String(v.Name)
	if v.Version != "" {
		w.RawString(",\"version\":")
		w.String(v.Version)
	}
	w.RawByte('}')
}

func (v *Runtime) MarshalFastJSON(w *fastjson.Writer) {
	w.RawByte('{')
	w.RawString("\"name\":")
	w.String(v.Name)
	w.RawString(",\"version\":")
	w.String(v.Version)
	w.RawByte('}')
}

func (v *System) MarshalFastJSON(w *fastjson.Writer) {
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
}

func (v *Process) MarshalFastJSON(w *fastjson.Writer) {
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
}

func (v *Transaction) MarshalFastJSON(w *fastjson.Writer) {
	w.RawByte('{')
	w.RawString("\"duration\":")
	w.Float64(v.Duration)
	w.RawString(",\"id\":")
	v.ID.MarshalFastJSON(w)
	w.RawString(",\"name\":")
	w.String(v.Name)
	w.RawString(",\"timestamp\":")
	v.Timestamp.MarshalFastJSON(w)
	w.RawString(",\"type\":")
	w.String(v.Type)
	if v.Context != nil {
		w.RawString(",\"context\":")
		v.Context.MarshalFastJSON(w)
	}
	if !v.ParentID.isZero() {
		w.RawString(",\"parent_id\":")
		v.ParentID.MarshalFastJSON(w)
	}
	if v.Result != "" {
		w.RawString(",\"result\":")
		w.String(v.Result)
	}
	if v.Sampled != nil {
		w.RawString(",\"sampled\":")
		w.Bool(*v.Sampled)
	}
	if !v.SpanCount.isZero() {
		w.RawString(",\"span_count\":")
		v.SpanCount.MarshalFastJSON(w)
	}
	if v.Spans != nil {
		w.RawString(",\"spans\":")
		w.RawByte('[')
		for i, v := range v.Spans {
			if i != 0 {
				w.RawByte(',')
			}
			v.MarshalFastJSON(w)
		}
		w.RawByte(']')
	}
	if !v.TraceID.isZero() {
		w.RawString(",\"trace_id\":")
		v.TraceID.MarshalFastJSON(w)
	}
	w.RawByte('}')
}

func (v *SpanCount) MarshalFastJSON(w *fastjson.Writer) {
	w.RawByte('{')
	if !v.Dropped.isZero() {
		w.RawString("\"dropped\":")
		v.Dropped.MarshalFastJSON(w)
	}
	w.RawByte('}')
}

func (v *SpanCountDropped) MarshalFastJSON(w *fastjson.Writer) {
	w.RawByte('{')
	w.RawString("\"total\":")
	w.Int64(int64(v.Total))
	w.RawByte('}')
}

func (v *Span) MarshalFastJSON(w *fastjson.Writer) {
	w.RawByte('{')
	w.RawString("\"duration\":")
	w.Float64(v.Duration)
	w.RawString(",\"name\":")
	w.String(v.Name)
	w.RawString(",\"start\":")
	w.Float64(v.Start)
	w.RawString(",\"type\":")
	w.String(v.Type)
	if v.Context != nil {
		w.RawString(",\"context\":")
		v.Context.MarshalFastJSON(w)
	}
	if !v.ID.isZero() {
		w.RawString(",\"id\":")
		v.ID.MarshalFastJSON(w)
	}
	if !v.ParentID.isZero() {
		w.RawString(",\"parent\":")
		v.ParentID.MarshalFastJSON(w)
	}
	if v.Stacktrace != nil {
		w.RawString(",\"stacktrace\":")
		w.RawByte('[')
		for i, v := range v.Stacktrace {
			if i != 0 {
				w.RawByte(',')
			}
			v.MarshalFastJSON(w)
		}
		w.RawByte(']')
	}
	if !v.TraceID.isZero() {
		w.RawString(",\"trace_id\":")
		v.TraceID.MarshalFastJSON(w)
	}
	w.RawByte('}')
}

func (v *SpanContext) MarshalFastJSON(w *fastjson.Writer) {
	w.RawByte('{')
	if v.Database != nil {
		w.RawString("\"db\":")
		v.Database.MarshalFastJSON(w)
	}
	w.RawByte('}')
}

func (v *DatabaseSpanContext) MarshalFastJSON(w *fastjson.Writer) {
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
}

func (v *Context) MarshalFastJSON(w *fastjson.Writer) {
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
		v.Custom.MarshalFastJSON(w)
	}
	if v.Request != nil {
		const prefix = ",\"request\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		v.Request.MarshalFastJSON(w)
	}
	if v.Response != nil {
		const prefix = ",\"response\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		v.Response.MarshalFastJSON(w)
	}
	if v.Tags != nil {
		const prefix = ",\"tags\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.RawByte('{')
		{
			first := true
			for k, v := range v.Tags {
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
	if v.User != nil {
		const prefix = ",\"user\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		v.User.MarshalFastJSON(w)
	}
	w.RawByte('}')
}

func (v *User) MarshalFastJSON(w *fastjson.Writer) {
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
	if !v.ID.isZero() {
		const prefix = ",\"id\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		v.ID.MarshalFastJSON(w)
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
}

func (v *Error) MarshalFastJSON(w *fastjson.Writer) {
	w.RawByte('{')
	w.RawString("\"timestamp\":")
	v.Timestamp.MarshalFastJSON(w)
	if v.Context != nil {
		w.RawString(",\"context\":")
		v.Context.MarshalFastJSON(w)
	}
	if v.Culprit != "" {
		w.RawString(",\"culprit\":")
		w.String(v.Culprit)
	}
	if !v.Exception.isZero() {
		w.RawString(",\"exception\":")
		v.Exception.MarshalFastJSON(w)
	}
	if v.ID != "" {
		w.RawString(",\"id\":")
		w.String(v.ID)
	}
	if !v.Log.isZero() {
		w.RawString(",\"log\":")
		v.Log.MarshalFastJSON(w)
	}
	if !v.ParentID.isZero() {
		w.RawString(",\"parent_id\":")
		v.ParentID.MarshalFastJSON(w)
	}
	if !v.TraceID.isZero() {
		w.RawString(",\"trace_id\":")
		v.TraceID.MarshalFastJSON(w)
	}
	if !v.Transaction.isZero() {
		w.RawString(",\"transaction\":")
		v.Transaction.MarshalFastJSON(w)
	}
	w.RawByte('}')
}

func (v *TransactionReference) MarshalFastJSON(w *fastjson.Writer) {
	w.RawByte('{')
	w.RawString("\"id\":")
	v.ID.MarshalFastJSON(w)
	w.RawByte('}')
}

func (v *Exception) MarshalFastJSON(w *fastjson.Writer) {
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
				fastjson.Marshal(w, v)
			}
		}
		w.RawByte('}')
	}
	if !v.Code.isZero() {
		w.RawString(",\"code\":")
		v.Code.MarshalFastJSON(w)
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
			v.MarshalFastJSON(w)
		}
		w.RawByte(']')
	}
	if v.Type != "" {
		w.RawString(",\"type\":")
		w.String(v.Type)
	}
	w.RawByte('}')
}

func (v *StacktraceFrame) MarshalFastJSON(w *fastjson.Writer) {
	w.RawByte('{')
	w.RawString("\"filename\":")
	w.String(v.File)
	w.RawString(",\"lineno\":")
	w.Int64(int64(v.Line))
	if v.AbsolutePath != "" {
		w.RawString(",\"abs_path\":")
		w.String(v.AbsolutePath)
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
				fastjson.Marshal(w, v)
			}
		}
		w.RawByte('}')
	}
	w.RawByte('}')
}

func (v *Log) MarshalFastJSON(w *fastjson.Writer) {
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
			v.MarshalFastJSON(w)
		}
		w.RawByte(']')
	}
	w.RawByte('}')
}

func (v *Request) MarshalFastJSON(w *fastjson.Writer) {
	w.RawByte('{')
	w.RawString("\"method\":")
	w.String(v.Method)
	w.RawString(",\"url\":")
	v.URL.MarshalFastJSON(w)
	if v.Body != nil {
		w.RawString(",\"body\":")
		v.Body.MarshalFastJSON(w)
	}
	if !v.Cookies.isZero() {
		w.RawString(",\"cookies\":")
		v.Cookies.MarshalFastJSON(w)
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
	if v.Headers != nil {
		w.RawString(",\"headers\":")
		v.Headers.MarshalFastJSON(w)
	}
	if v.HTTPVersion != "" {
		w.RawString(",\"http_version\":")
		w.String(v.HTTPVersion)
	}
	if v.Socket != nil {
		w.RawString(",\"socket\":")
		v.Socket.MarshalFastJSON(w)
	}
	w.RawByte('}')
}

func (v *RequestHeaders) MarshalFastJSON(w *fastjson.Writer) {
	w.RawByte('{')
	first := true
	if v.ContentType != "" {
		const prefix = ",\"content-type\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.ContentType)
	}
	if v.Cookie != "" {
		const prefix = ",\"cookie\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.Cookie)
	}
	if v.UserAgent != "" {
		const prefix = ",\"user-agent\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		w.String(v.UserAgent)
	}
	w.RawByte('}')
}

func (v *RequestSocket) MarshalFastJSON(w *fastjson.Writer) {
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
}

func (v *Response) MarshalFastJSON(w *fastjson.Writer) {
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
	if v.Headers != nil {
		const prefix = ",\"headers\":"
		if first {
			first = false
			w.RawString(prefix[1:])
		} else {
			w.RawString(prefix)
		}
		v.Headers.MarshalFastJSON(w)
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
}

func (v *ResponseHeaders) MarshalFastJSON(w *fastjson.Writer) {
	w.RawByte('{')
	if v.ContentType != "" {
		w.RawString("\"content-type\":")
		w.String(v.ContentType)
	}
	w.RawByte('}')
}

func (v *Metrics) MarshalFastJSON(w *fastjson.Writer) {
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
				v.MarshalFastJSON(w)
			}
		}
		w.RawByte('}')
	}
	w.RawString(",\"timestamp\":")
	v.Timestamp.MarshalFastJSON(w)
	if !v.Labels.isZero() {
		w.RawString(",\"labels\":")
		v.Labels.MarshalFastJSON(w)
	}
	w.RawByte('}')
}

func (v *Metric) MarshalFastJSON(w *fastjson.Writer) {
	w.RawByte('{')
	w.RawString("\"type\":")
	w.String(v.Type)
	if v.Count != nil {
		w.RawString(",\"count\":")
		w.Uint64(*v.Count)
	}
	if v.Max != nil {
		w.RawString(",\"max\":")
		w.Float64(*v.Max)
	}
	if v.Min != nil {
		w.RawString(",\"min\":")
		w.Float64(*v.Min)
	}
	if v.Quantiles != nil {
		w.RawString(",\"quantiles\":")
		w.RawByte('[')
		for i, v := range v.Quantiles {
			if i != 0 {
				w.RawByte(',')
			}
			v.MarshalFastJSON(w)
		}
		w.RawByte(']')
	}
	if v.Stddev != nil {
		w.RawString(",\"stddev\":")
		w.Float64(*v.Stddev)
	}
	if v.Sum != nil {
		w.RawString(",\"sum\":")
		w.Float64(*v.Sum)
	}
	if v.Unit != "" {
		w.RawString(",\"unit\":")
		w.String(v.Unit)
	}
	if v.Value != nil {
		w.RawString(",\"value\":")
		w.Float64(*v.Value)
	}
	w.RawByte('}')
}

func (v *TransactionsPayload) MarshalFastJSON(w *fastjson.Writer) {
	w.RawByte('{')
	w.RawString("\"service\":")
	if v.Service == nil {
		w.RawString("null")
	} else {
		v.Service.MarshalFastJSON(w)
	}
	w.RawString(",\"transactions\":")
	if v.Transactions == nil {
		w.RawString("null")
	} else {
		w.RawByte('[')
		for i, v := range v.Transactions {
			if i != 0 {
				w.RawByte(',')
			}
			v.MarshalFastJSON(w)
		}
		w.RawByte(']')
	}
	if v.Process != nil {
		w.RawString(",\"process\":")
		v.Process.MarshalFastJSON(w)
	}
	if v.System != nil {
		w.RawString(",\"system\":")
		v.System.MarshalFastJSON(w)
	}
	w.RawByte('}')
}

func (v *ErrorsPayload) MarshalFastJSON(w *fastjson.Writer) {
	w.RawByte('{')
	w.RawString("\"errors\":")
	if v.Errors == nil {
		w.RawString("null")
	} else {
		w.RawByte('[')
		for i, v := range v.Errors {
			if i != 0 {
				w.RawByte(',')
			}
			v.MarshalFastJSON(w)
		}
		w.RawByte(']')
	}
	w.RawString(",\"service\":")
	if v.Service == nil {
		w.RawString("null")
	} else {
		v.Service.MarshalFastJSON(w)
	}
	if v.Process != nil {
		w.RawString(",\"process\":")
		v.Process.MarshalFastJSON(w)
	}
	if v.System != nil {
		w.RawString(",\"system\":")
		v.System.MarshalFastJSON(w)
	}
	w.RawByte('}')
}

func (v *MetricsPayload) MarshalFastJSON(w *fastjson.Writer) {
	w.RawByte('{')
	w.RawString("\"metrics\":")
	if v.Metrics == nil {
		w.RawString("null")
	} else {
		w.RawByte('[')
		for i, v := range v.Metrics {
			if i != 0 {
				w.RawByte(',')
			}
			v.MarshalFastJSON(w)
		}
		w.RawByte(']')
	}
	if v.Process != nil {
		w.RawString(",\"process\":")
		v.Process.MarshalFastJSON(w)
	}
	if v.Service != nil {
		w.RawString(",\"service\":")
		v.Service.MarshalFastJSON(w)
	}
	if v.System != nil {
		w.RawString(",\"system\":")
		v.System.MarshalFastJSON(w)
	}
	w.RawByte('}')
}
