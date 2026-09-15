package wallets

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// writeJSON escribe una respuesta JSON con status.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// writeErr escribe un error como JSON {"error": "..."}.
func writeErr(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

// decodeJSON decodifica el cuerpo de la request en v (DTO del borde).
func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("json inválido: %w", err)
	}
	return nil
}
