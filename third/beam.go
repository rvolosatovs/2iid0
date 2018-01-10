package main

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	_ "net/http/pprof"
	"os"
	"reflect"
	"sort"
)

type condition struct {
	attribute string
}

func (c condition) Attribute() string {
	return c.attribute
}

type Attributor interface {
	fmt.Stringer
	Attribute() string
}

type booleanCondition struct {
	condition
	value bool
}

func (c booleanCondition) String() string {
	return fmt.Sprintf("%s == %v", c.attribute, c.value)
}

type nominalCondition struct {
	condition
	value string
	equal bool
}

func (c nominalCondition) String() string {
	sign := "!="
	if c.equal {
		sign = "=="
	}
	return fmt.Sprintf("%s %s %v", c.attribute, sign, c.value)
}

type numericCondition struct {
	condition
	value   float64
	greater bool
}

func (c numericCondition) String() string {
	sign := "<="
	if c.greater {
		sign = ">="
	}
	return fmt.Sprintf("%s %s %v", c.attribute, sign, c.value)
}

type description []Attributor

func (d description) String() string {
	if len(d) == 0 {
		return ""
	}
	out := d[0].String()
	for i := 1; i < len(d); i++ {
		out = fmt.Sprintf("%s AND %s", out, d[i])
	}
	return out
}

type Item struct {
	value   description
	quality float64
}

// A PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue struct {
	queue []*Item
	limit int
}

func (pq PriorityQueue) Len() int { return len(pq.queue) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, quality so we use greater than here.
	return pq.queue[i].quality > pq.queue[j].quality
}

func (pq PriorityQueue) Swap(i, j int) {
	pq.queue[i], pq.queue[j] = pq.queue[j], pq.queue[i]
}

func (pq *PriorityQueue) Push(x interface{}) {
	item := x.(*Item)
	if len(pq.queue) == pq.limit {
		if item.quality < pq.queue[0].quality {
			return
		}
		heap.Remove(pq, 0)
	}
	pq.queue = append(pq.queue, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	n := len(pq.queue)
	item := pq.queue[n-1]
	pq.queue = pq.queue[:n-1]
	return item
}

func NewPriorityQueue(limit int) (pq *PriorityQueue) {
	defer func() {
		heap.Init(pq)
	}()
	return &PriorityQueue{
		limit: limit,
		queue: make([]*Item, 0, limit),
	}
}

func beamSearch(qm func(description) float64, ro func(description) []description, w, d, q int) (rs *PriorityQueue) {
	cq := make([]description, 0, w)
	cq = append(cq, description{})

	rs = NewPriorityQueue(q)
	for i := 0; i < d; i++ {
		b := NewPriorityQueue(w)
		for _, seed := range cq {
			for _, desc := range ro(seed) {
				item := &Item{
					value:   desc,
					quality: qm(desc),
				}
				heap.Push(b, item)
				heap.Push(rs, item)
			}

			n := b.Len()
			cq = make([]description, n)
			for j := 0; j < n; j++ {
				cq[j] = heap.Pop(b).(*Item).value
			}
		}
	}
	return rs
}

type Row struct {
	//Index         *float64 `json:"index"`
	//GeoCountry    *string `json:"geo_country"`
	GeoRegion *string `json:"geo_region"`
	//GeoCity       *string `json:"geo_city"`
	//GeoRegionName *string `json:"geo_region_name"`
	//GeoTimezone   *string `json:"geo_timezone"`
	//PageUrl           *string  `json:"page_url"`
	//PageTitle         *string  `json:"page_title"`
	//PageReferrer      *string  `json:"page_referrer"`
	RefrSource *string `json:"refr_source"`
	//PpXoffsetMin      *float64 `json:"pp_xoffset_min"`
	//PpXoffsetMax      *float64 `json:"pp_xoffset_max"`
	//PpYoffsetMin      *float64 `json:"pp_yoffset_min"`
	//PpYoffsetMax      *float64 `json:"pp_yoffset_max"`
	Useragent         *string  `json:"useragent"`
	BrowserLanguage   *string  `json:"browser_language"`
	BrowserCookies    *bool    `json:"browser_cookies"`
	BrowserColordepth *float64 `json:"browser_colordepth"`
	BrowserViewdepth  *float64 `json:"browser_viewdepth"`
	BrowserViewheight *float64 `json:"browser_viewheight"`
	OsName            *string  `json:"os_name"`
	//OsTimezone        *string  `json:"os_timezone"`
	DvceType *string `json:"dvce_type"`
	//ActionLabel       *string  `json:"action_label"`
	//ActionType        *string  `json:"action_type"`
	Condition1 *bool `json:"condition_1"`
	Clicked    *bool `json:"clicked"`
}

func (r *Row) field(name string) reflect.Value {
	return rowField(r, name)
}

func (r *Row) matches(d description) bool {
	ok := true
	for _, c := range d {
		f := r.field(c.Attribute())

		if f.IsNil() {
			return false
		}
		f = f.Elem()

		switch c := c.(type) {
		case booleanCondition:
			if c.value != f.Bool() {
				return false
			}
		case nominalCondition:
			if c.equal {
				if c.value != f.String() {
					return false
				}
			} else {
				if c.value == f.String() {
					return false
				}
			}
		case numericCondition:
			if c.greater {
				if c.value <= f.Float() {
					return false
				}
			} else {
				if c.value >= f.Float() {
					return false
				}
			}
		}
	}
	return ok
}

var rowType = reflect.TypeOf((*Row)(nil)).Elem()
var rowField func(r *Row, name string) reflect.Value

func main() {
	go func() {
		if err := http.ListenAndServe(":6060", nil); err != nil {
			log.Fatalf("Failed to run pprof endpoint: %s", err)
		}
	}()
	data := struct {
		Rows    []*Row          `json:"rows"`
		Width   int             `json:"width"`
		Depth   int             `json:"depth"`
		Results int             `json:"results"`
		Bins    int             `json:"bins"`
		Targets map[string]bool `json:"targets"`
	}{}
	log.Println("Decoding data...")
	if err := json.NewDecoder(os.Stdin).Decode(&data); err != nil {
		log.Fatal(err)
	}

	rowIdx := make(map[string]int)
	for i := 0; i < rowType.NumField(); i++ {
		rowIdx[rowType.Field(i).Name] = i
	}

	rowFields := make(map[*Row][]reflect.Value)
	for _, r := range data.Rows {
		rv := reflect.ValueOf(r).Elem()

		vs := make([]reflect.Value, rowType.NumField())
		for i := range vs {
			vs[i] = rv.Field(i)
		}
		rowFields[r] = vs
	}

	rowField = func(r *Row, name string) reflect.Value {
		return rowFields[r][rowIdx[name]]
	}

	log.Printf("Starting beam search on %d rows of %d columns with %d targets....", len(data.Rows), rowType.NumField(), len(data.Targets))
	log.Printf("Width: %d", data.Width)
	log.Printf("Depth: %d", data.Depth)
	log.Printf("Results: %d", data.Results)
	log.Printf("Bins: %d", data.Bins)

	matches := make(map[string][]*Row)
	nomatches := make(map[string][]*Row)
	rowsMatching := func(d description) []*Row {
		s := d.String()

		rows, ok := matches[s]
		if ok {
			return rows
		}

		rows = make([]*Row, 0, len(data.Rows)/10000)
		for _, r := range data.Rows {
			if r.matches(d) {
				rows = append(rows, r)
			}
		}
		matches[s] = rows
		return rows
	}

	q := beamSearch(
		func(d description) float64 {
			// Quality measure

			s := d.String()

			matching, mok := matches[s]
			nomatching, nok := nomatches[s]
			if !mok || !nok {
				matching = make([]*Row, 0, len(data.Rows)/10000)
				nomatching = make([]*Row, 0, len(data.Rows)/10000)
				for _, r := range data.Rows {
					if r.matches(d) {
						matching = append(matching, r)
					} else {
						nomatching = append(nomatching, r)
					}
				}
				matches[s] = matching
				nomatches[s] = nomatching
			}

			n1, n2, n3, n4 := 0., 0., 0., 0.
			for _, r := range matching {
				switch {
				case *r.Condition1 && !*r.Clicked:
					n1++
				case *r.Condition1 && *r.Clicked:
					n2++
				case !*r.Condition1 && !*r.Clicked:
					n3++
				case !*r.Condition1 && *r.Clicked:
					n4++
				}
			}
			qs := (n1*n4 - n2*n3) / (n1*n4 + n2*n3)

			n1, n2, n3, n4 = 0., 0., 0., 0.
			for _, r := range nomatching {
				switch {
				case *r.Condition1 && !*r.Clicked:
					n1++
				case *r.Condition1 && *r.Clicked:
					n2++
				case !*r.Condition1 && !*r.Clicked:
					n3++
				case !*r.Condition1 && *r.Clicked:
					n4++
				}
			}
			qsc := (n1*n4 - n2*n3) / (n1*n4 + n2*n3)

			qy := qs - qsc
			if qy < 0 {
				qy *= -1
			}

			ndiv := float64(len(matching)) / float64(len(data.Rows))
			nbardiv := float64(len(nomatching)) / float64(len(data.Rows))
			qef := -ndiv*math.Log(ndiv) - nbardiv*math.Log(nbardiv)

			return qy * qef
		},
		func(d description) []description {
			// Refinement operator

			attrs := make(map[string]struct{}, rowType.NumField())
			for i := 0; i < rowType.NumField(); i++ {
				attrs[rowType.Field(i).Name] = struct{}{}
			}
			for _, c := range d {
				delete(attrs, c.Attribute())
			}
			for name := range data.Targets {
				delete(attrs, name)
			}

			rows := rowsMatching(d)

			out := make([]description, 0, len(d)*2)
			for name := range attrs {
				var cs []Attributor
				cond := condition{
					attribute: name,
				}

				f, ok := rowType.FieldByName(name)
				if !ok {
					panic(fmt.Errorf("Column %s not present in row", f.Name))
				}
				switch f.Type.Elem().Kind() {
				case reflect.Bool:
					cs = append(cs,
						booleanCondition{cond, true},
						booleanCondition{cond, false},
					)
				case reflect.String:
					uniq := make(map[string]struct{}, len(rows)/2)
					for _, r := range rows {
						rv := r.field(name)
						if rv.IsNil() {
							continue
						}
						uniq[rv.Elem().String()] = struct{}{}
					}
					for name := range uniq {
						cs = append(cs,
							nominalCondition{cond, name, true},
							nominalCondition{cond, name, false},
						)
					}
				case reflect.Float64:
					uniq := make(map[float64]struct{}, len(rows)/2)
					for _, r := range rows {
						rv := r.field(name)
						if rv.IsNil() {
							continue
						}
						uniq[rv.Elem().Float()] = struct{}{}
					}

					vals := make([]float64, 0, len(uniq))
					for v := range uniq {
						vals = append(vals, v)
					}
					sort.Float64s(vals)

					n := data.Bins
					if len(vals)/n < 1 {
						n = len(vals)
					}
					for k := 1; k < n-1; k++ {
						cs = append(cs,
							numericCondition{cond, vals[k], true},
							numericCondition{cond, vals[k], false},
						)
					}
				default:
					panic(fmt.Errorf("Unmatched kind for column %s: %s", name, f.Type.Kind()))
				}

				for _, c := range cs {
					out = append(out, append(append(make(description, 0, len(d)+1), d...), c))
				}
			}
			return out
		},
		data.Width,
		data.Depth,
		data.Results,
	)

	for q.Len() > 0 {
		fmt.Println(heap.Pop(q).(*Item).value)
	}
}
