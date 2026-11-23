package repositories

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/models"
)

func TestSecurityRepository_InterfaceConformance(t *testing.T) {
	var _ SecurityRepository = (*securityRepository)(nil)
	var _ SecurityRepository = (*MockSecurityRepository)(nil)

	repo := NewSecurityRepository(nil)
	if repo == nil {
		t.Fatal("NewSecurityRepository mengembalikan nil")
	}
}

func TestCreateAuditLog_DefaultsCategoryAndSeverity(t *testing.T) {
	ctx := context.Background()
	var capturedArgs []driver.Value

	state := &stubDBState{
		execRouter: func(query string, args []driver.Value) stubExecResult {
			capturedArgs = args
			return stubExecResult{affected: 1}
		},
	}
	repo := NewAdminUserRepository(newStubDB(t, state))

	err := repo.CreateAuditLog(ctx, &models.AuditLog{
		ActorID:        "staff-1",
		ActorName:      "Super Admin",
		ActorRole:      "SUPER_ADMIN",
		Action:         "CREATE_STAFF",
		TargetResource: "staff:petugas1",
		Details:        "Membuat akun petugas",
		IPAddress:      "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("CreateAuditLog gagal: %v", err)
	}
	if len(capturedArgs) != 13 {
		t.Fatalf("Ekspektasi 13 argumen INSERT, dapat %d", len(capturedArgs))
	}
	if capturedArgs[8] != "OPERATIONAL" {
		t.Fatalf("Ekspektasi default category OPERATIONAL, dapat %v", capturedArgs[8])
	}
	if capturedArgs[9] != "INFO" {
		t.Fatalf("Ekspektasi default severity INFO, dapat %v", capturedArgs[9])
	}
	if capturedArgs[10] != int64(200) {
		t.Fatalf("Ekspektasi default statusCode 200, dapat %v", capturedArgs[10])
	}
}

func TestGetAuditLogsFiltered_AppliesFiltersAndClampsLimit(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	from := now.Add(-24 * time.Hour)
	to := now

	state := &stubDBState{
		queryRouter: func(query string, args []driver.Value) stubQueryResult {
			if strings.Contains(query, "SELECT COUNT(*)") {
				return stubQueryResult{
					columns: []string{"count"},
					rows:    [][]driver.Value{{int64(1)}},
				}
			}
			return stubQueryResult{
				columns: []string{
					"id", "actor_id", "actor_name", "actor_role", "action", "target_resource",
					"details", "ip_address", "category", "severity", "status_code", "user_agent", "created_at",
				},
				rows: [][]driver.Value{{
					"audit-sec-1", "anonymous", "10.0.0.5", "ANONYMOUS", "FAILED_AUTH", "POST /api/v1/auth/login/password",
					"HTTP 401", "10.0.0.5", "AUTH", "WARN", int64(401), "curl/8.0", now,
				}},
			}
		},
	}

	repo := NewSecurityRepository(newStubDB(t, state))
	page, err := repo.GetAuditLogsFiltered(ctx, models.AuditLogFilter{
		Category:  "AUTH",
		Severity:  "WARN",
		ActorRole: "ANONYMOUS",
		IPAddress: "10.0.0.5",
		Search:    "login",
		From:      &from,
		To:        &to,
		Page:      0,
		Limit:     999,
	})
	if err != nil {
		t.Fatalf("GetAuditLogsFiltered gagal: %v", err)
	}
	if page.Page != 1 || page.Limit != 200 || page.Total != 1 {
		t.Fatalf("Pagination tidak sesuai: %+v", page)
	}
	if len(page.Items) != 1 || page.Items[0].Severity != "WARN" || page.Items[0].StatusCode != 401 {
		t.Fatalf("Item audit tidak sesuai: %+v", page.Items)
	}
	if len(state.calls) < 2 {
		t.Fatalf("Ekspektasi 2 query (COUNT + SELECT), dapat %d", len(state.calls))
	}
	selectQuery := state.calls[1].query
	for _, fragment := range []string{"UPPER(category) =", "UPPER(severity) =", "UPPER(actor_role) =", "ip_address LIKE", "created_at >=", "created_at <="} {
		if !strings.Contains(selectQuery, fragment) {
			t.Fatalf("Query SELECT tidak menyertakan filter %q: %s", fragment, selectQuery)
		}
	}
}

func TestBlockAndUnblockIP_HandlesExpiryAndNotFound(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	future := now.Add(15 * time.Minute)

	state := &stubDBState{
		execRouter: func(query string, args []driver.Value) stubExecResult {
			if strings.Contains(query, "SET is_active = FALSE") && len(args) == 1 && args[0] == "not-exist" {
				return stubExecResult{affected: 0}
			}
			return stubExecResult{affected: 1}
		},
		queryRouter: func(query string, args []driver.Value) stubQueryResult {
			if len(args) > 0 && args[0] == "10.0.0.99" {
				return stubQueryResult{
					columns: []string{"id", "ip_address", "reason", "blocked_by", "is_auto", "is_active", "expires_at", "created_at"},
					rows: [][]driver.Value{{
						"blk-1", "10.0.0.99", "Brute force", "SYSTEM_AUTO_DEFENSE", true, true, future, now,
					}},
				}
			}
			return stubQueryResult{err: sql.ErrNoRows}
		},
	}

	repo := NewSecurityRepository(newStubDB(t, state))

	blk, err := repo.BlockIP(ctx, &models.BlockedIP{
		IPAddress: "10.0.0.99",
		Reason:    "Brute force",
		BlockedBy: "SYSTEM_AUTO_DEFENSE",
		IsAuto:    true,
		ExpiresAt: &future,
	})
	if err != nil {
		t.Fatalf("BlockIP gagal: %v", err)
	}
	if !blk.IsActive || blk.IPAddress != "10.0.0.99" {
		t.Fatalf("Hasil BlockIP tidak sesuai: %+v", blk)
	}

	found, err := repo.IsIPBlocked(ctx, "10.0.0.99")
	if err != nil || found == nil || found.ID != "blk-1" {
		t.Fatalf("IsIPBlocked gagal: found=%+v err=%v", found, err)
	}

	_, err = repo.IsIPBlocked(ctx, "10.0.0.1")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Ekspektasi ErrNotFound untuk IP tidak diblokir, dapat %v", err)
	}

	if err := repo.UnblockIP(ctx, "not-exist"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Ekspektasi ErrNotFound saat unblock IP tidak ada, dapat %v", err)
	}

	if err := repo.UnblockIP(ctx, "10.0.0.99"); err != nil {
		t.Fatalf("UnblockIP gagal: %v", err)
	}
}
