package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"assessment.local/returns/internal/domain"
	"assessment.local/returns/internal/service"
)

type Source interface {
	Read(context.Context) (domain.Dataset, error)
	Ping(context.Context) error
}

type Server struct{ source Source }

func New(source Source) http.Handler {
	s := &Server{source: source}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		if err := s.source.Ping(ctx); err != nil {
			writeError(w, http.StatusServiceUnavailable, "База данных временно недоступна")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/meta", s.withDataset(s.meta))
	mux.HandleFunc("GET /api/operations", s.withDataset(s.operationReport))
	mux.HandleFunc("GET /api/returns", s.withDataset(s.returnList))
	mux.HandleFunc("GET /api/returns/{id}", s.withDataset(s.returnDetail))
	mux.HandleFunc("GET /api/returns-report", s.withDataset(s.returnReport))
	mux.HandleFunc("GET /api/snapshot", s.withDataset(func(w http.ResponseWriter, r *http.Request, data domain.Dataset) { writeJSON(w, http.StatusOK, data) }))
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Маршрут не найден"})
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		mux.ServeHTTP(w, r)
	})
}

func (s *Server) withDataset(handler func(http.ResponseWriter, *http.Request, domain.Dataset)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		data, err := s.source.Read(ctx)
		if err != nil {
			log.Printf("read dataset: %v", err)
			writeError(w, http.StatusServiceUnavailable, "Не удалось получить данные из базы. Повторите запрос.")
			return
		}
		handler(w, r.WithContext(ctx), data)
	}
}

func (s *Server) meta(w http.ResponseWriter, r *http.Request, data domain.Dataset) {
	writeJSON(w, http.StatusOK, map[string]any{"manifest": data.Manifest, "suppliers": data.Suppliers})
}

func (s *Server) filter(r *http.Request, manifest domain.Manifest) (service.Filter, error) {
	q := r.URL.Query()
	from, to := q.Get("from"), q.Get("to")
	if from == "" {
		from = manifest.From
	}
	if to == "" {
		to = manifest.To
	}
	return service.NewFilter(from, to, q.Get("supplier_id"), q.Get("q"))
}

func (s *Server) operationReport(w http.ResponseWriter, r *http.Request, data domain.Dataset) {
	filter, err := s.filter(r, data.Manifest)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, service.NewOperations(service.DatasetSource(data)).Report(filter))
}

func (s *Server) returnReport(w http.ResponseWriter, r *http.Request, data domain.Dataset) {
	filter, err := s.filter(r, data.Manifest)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	snapshotAt, err := time.Parse(time.RFC3339, data.Manifest.AsOf)
	if err != nil {
		log.Printf("parse dataset as_of: %v", err)
		writeError(w, http.StatusInternalServerError, "Некорректная дата среза данных")
		return
	}

	// Until — начало следующего дня; StockAt использует включительный срез.
	cutoff := filter.Until.Add(-time.Nanosecond)
	if cutoff.After(snapshotAt) {
		cutoff = snapshotAt
	}
	if filter.From.After(cutoff) {
		writeError(w, http.StatusBadRequest, "Выбранный период находится после доступного среза данных")
		return
	}

	priorities := service.ReturnPriorities(data, cutoff, filter.SupplierID)
	writeJSON(w, http.StatusOK, map[string]any{
		"as_of":                cutoff.Format(time.RFC3339Nano),
		"stock":                service.StockAt(data, cutoff, filter.SupplierID),
		"days":                 service.StockTrend(data, filter.From, cutoff, filter.SupplierID),
		"suppliers":            service.StockBySupplier(data, cutoff, filter.SupplierID),
		"priorities":           priorities,
		"total_priority_cases": len(priorities),
		"priority_rules": map[string]any{
			"urgent_window_days": service.UrgentReturnWindowDays,
			"order":              "Сначала срочные случаи, затем сумма остатка по убыванию, возраст по убыванию и идентификатор.",
		},
		"limitations": []string{
			"Очередь включает только товары с положительным остатком на нашем складе. Отправленные товары без складского остатка сюда не входят.",
			"Срок относится к получению поставщиком. Время доставки неизвестно, поэтому открытое окно не гарантирует своевременное получение.",
			"Сумма количеств по отказам относится к отправкам и может включать повторные попытки. Это не количество уникальных единиц текущего остатка.",
			"История изменений качества, комплектности и подтверждения условий отсутствует; для прошлых дат используются сведения из учебного набора.",
		},
	})
}

func (s *Server) returnList(w http.ResponseWriter, r *http.Request, data domain.Dataset) {
	filter, err := s.filter(r, data.Manifest)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	page, pageSize := 1, 25
	for key, target := range map[string]*int{"page": &page, "page_size": &pageSize} {
		if value := r.URL.Query().Get(key); value != "" {
			parsed, err := strconv.Atoi(value)
			if err != nil || parsed < 1 || parsed > 10000 || (key == "page_size" && parsed > 100) {
				writeError(w, http.StatusBadRequest, "Некорректные параметры страницы")
				return
			}
			*target = parsed
		}
	}
	writeJSON(w, http.StatusOK, service.NewRegister(service.DatasetSource(data)).List(filter, (page-1)*pageSize, pageSize))
}

func (s *Server) returnDetail(w http.ResponseWriter, r *http.Request, data domain.Dataset) {
	detail, found := service.NewRegister(service.DatasetSource(data)).Detail(r.PathValue("id"))
	if !found {
		writeError(w, http.StatusNotFound, "Возврат не найден")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
