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
	"github.com/lib/pq"
)

func TestAdminUserRepository_InterfaceConformance(t *testing.T) {
	var _ AdminUserRepository = (*adminUserRepository)(nil)
	var _ AdminUserRepository = (*MockAdminUserRepository)(nil)

	repo := NewAdminUserRepository(nil)
	if repo == nil {
		t.Fatal("NewAdminUserRepository mengembalikan nil")
	}
}

func adminUserColumns() []string {
	return []string{
		"id", "username", "name", "email", "password_hash",
		"role", "is_active", "created_at", "updated_at",
	}
}

func TestAdminUserRepository_QueriesAndMutations(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("GetByUsername mengembalikan data staf dan ErrNotFound jika tidak ada", func(t *testing.T) {
		state := &stubDBState{
			queryRouter: func(query string, args []driver.Value) stubQueryResult {
				if len(args) > 0 && args[0] == "superadmin" {
					return stubQueryResult{
						columns: adminUserColumns(),
						rows: [][]driver.Value{{
							"staff-1", "superadmin", "Super Admin", "superadmin@symphoniatic.id",
							"$2a$10$hash", "SUPER_ADMIN", true, now, now,
						}},
					}
				}
				return stubQueryResult{err: sql.ErrNoRows}
			},
		}
		repo := NewAdminUserRepository(newStubDB(t, state))

		u, err := repo.GetByUsername(ctx, "superadmin")
		if err != nil {
			t.Fatalf("GetByUsername gagal: %v", err)
		}
		if u.Role != "SUPER_ADMIN" || !u.IsActive {
			t.Fatalf("Data staf tidak sesuai: %+v", u)
		}

		_, err = repo.GetByUsername(ctx, "tidakada")
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("Ekspektasi ErrNotFound, dapat: %v", err)
		}
	})

	t.Run("GetAllStaff memfilter role dan search dengan benar", func(t *testing.T) {
		state := &stubDBState{
			queryRouter: func(query string, args []driver.Value) stubQueryResult {
				return stubQueryResult{
					columns: adminUserColumns(),
					rows: [][]driver.Value{{
						"staff-2", "petugas1", "Petugas Gate 1", "petugas1@symphoniatic.id",
						"hash", "PETUGAS", true, now, now,
					}},
				}
			},
		}
		repo := NewAdminUserRepository(newStubDB(t, state))
		list, err := repo.GetAllStaff(ctx, "PETUGAS", "gate")
		if err != nil {
			t.Fatalf("GetAllStaff gagal: %v", err)
		}
		if len(list) != 1 || list[0].Username != "petugas1" {
			t.Fatalf("Hasil GetAllStaff tidak sesuai: %+v", list)
		}
		if len(state.calls) == 0 || !strings.Contains(state.calls[0].query, "AND role = $1") {
			t.Fatalf("Query tidak menyertakan filter role: %v", state.calls)
		}
	})

	t.Run("CreateStaff mendeteksi duplikat username/email", func(t *testing.T) {
		state := &stubDBState{
			execRouter: func(query string, args []driver.Value) stubExecResult {
				return stubExecResult{err: &pq.Error{Code: "23505"}}
			},
		}
		repo := NewAdminUserRepository(newStubDB(t, state))
		_, err := repo.CreateStaff(ctx, &models.CreateStaffRequest{
			Username: "admin",
			Name:     "Admin",
			Email:    "admin@symphoniatic.id",
			Role:     "ADMIN",
		}, "hashed")
		if !errors.Is(err, ErrStaffDuplicateUsername) {
			t.Fatalf("Ekspektasi ErrStaffDuplicateUsername, dapat: %v", err)
		}
	})

	t.Run("CreateAuditLog dan GetAuditLogs berfungsi dengan baik", func(t *testing.T) {
		state := &stubDBState{
			queryRouter: func(query string, args []driver.Value) stubQueryResult {
				return stubQueryResult{
					columns: []string{"id", "actor_id", "actor_name", "actor_role", "action", "target_resource", "details", "ip_address", "created_at"},
					rows: [][]driver.Value{{
						"audit-1", "staff-1", "Super Admin", "SUPER_ADMIN", "CREATE_STAFF", "staff:petugas1", "Membuat akun petugas baru", "127.0.0.1", now,
					}},
				}
			},
		}
		repo := NewAdminUserRepository(newStubDB(t, state))

		err := repo.CreateAuditLog(ctx, &models.AuditLog{
			ActorID:        "staff-1",
			ActorName:      "Super Admin",
			ActorRole:      "SUPER_ADMIN",
			Action:         "CREATE_STAFF",
			TargetResource: "staff:petugas1",
			Details:        "Membuat akun petugas baru",
			IPAddress:      "127.0.0.1",
		})
		if err != nil {
			t.Fatalf("CreateAuditLog gagal: %v", err)
		}

		logs, err := repo.GetAuditLogs(ctx, "CREATE_STAFF", "petugas", 50)
		if err != nil {
			t.Fatalf("GetAuditLogs gagal: %v", err)
		}
		if len(logs) != 1 || logs[0].Action != "CREATE_STAFF" {
			t.Fatalf("Hasil GetAuditLogs tidak sesuai: %+v", logs)
		}
	})

	t.Run("GetByID, UpdateStaff, ToggleStaffActive, ResetStaffPassword, DeleteStaff, dan CountActiveSuperAdmins", func(t *testing.T) {
		state := &stubDBState{
			queryRouter: func(query string, args []driver.Value) stubQueryResult {
				if strings.Contains(query, "SELECT COUNT(*)") {
					return stubQueryResult{
						columns: []string{"count"},
						rows:    [][]driver.Value{{int64(2)}},
					}
				}
				if strings.Contains(query, "SET is_active = NOT is_active") {
					return stubQueryResult{
						columns: adminUserColumns(),
						rows: [][]driver.Value{{
							"staff-2", "petugas1", "Petugas Gate", "p1@symphoniatic.id",
							"hash", "PETUGAS", false, now, now,
						}},
					}
				}
				if len(args) > 0 && args[0] == "missing" {
					return stubQueryResult{err: sql.ErrNoRows}
				}
				return stubQueryResult{
					columns: adminUserColumns(),
					rows: [][]driver.Value{{
						"staff-2", "petugas1", "Petugas Updated", "updated@symphoniatic.id",
						"hash", "ADMIN", true, now, now,
					}},
				}
			},
			execRouter: func(query string, args []driver.Value) stubExecResult {
				if len(args) > 0 && args[len(args)-1] == "missing" {
					return stubExecResult{affected: 0}
				}
				return stubExecResult{affected: 1}
			},
		}
		repo := NewAdminUserRepository(newStubDB(t, state))

		u, err := repo.GetByID(ctx, "staff-2")
		if err != nil || u.ID != "staff-2" {
			t.Fatalf("GetByID gagal: %v", err)
		}
		if _, err := repo.GetByID(ctx, "missing"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("GetByID untuk missing seharusnya ErrNotFound, dapat %v", err)
		}

		upd, err := repo.UpdateStaff(ctx, "staff-2", &models.UpdateStaffRequest{
			Name:  "Petugas Updated",
			Email: "updated@symphoniatic.id",
			Role:  "ADMIN",
		})
		if err != nil || upd.Role != "ADMIN" {
			t.Fatalf("UpdateStaff gagal: %v", err)
		}

		toggled, err := repo.ToggleStaffActive(ctx, "staff-2")
		if err != nil || toggled.IsActive {
			t.Fatalf("ToggleStaffActive gagal: err=%v active=%v", err, toggled)
		}

		if err := repo.ResetStaffPassword(ctx, "staff-2", "newhash"); err != nil {
			t.Fatalf("ResetStaffPassword gagal: %v", err)
		}
		if err := repo.ResetStaffPassword(ctx, "missing", "newhash"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("ResetStaffPassword missing seharusnya ErrNotFound, dapat %v", err)
		}

		if err := repo.DeleteStaff(ctx, "staff-2"); err != nil {
			t.Fatalf("DeleteStaff gagal: %v", err)
		}
		if err := repo.DeleteStaff(ctx, "missing"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("DeleteStaff missing seharusnya ErrNotFound, dapat %v", err)
		}

		cnt, err := repo.CountActiveSuperAdmins(ctx)
		if err != nil || cnt != 2 {
			t.Fatalf("CountActiveSuperAdmins gagal: err=%v cnt=%d", err, cnt)
		}
	})
}
