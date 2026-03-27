package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/fitreg/api/models"
)

// ErrMaxCoachesReached is returned when a student already has the maximum number of coaches.
var ErrMaxCoachesReached = errors.New("student has reached the maximum number of coaches")

type invitationRepository struct {
	db *sql.DB
}

func NewInvitationRepository(db *sql.DB) InvitationRepository {
	return &invitationRepository{db: db}
}

func (r *invitationRepository) GetStatus(id int64) (string, error) {
	var status string
	err := r.db.QueryRow("SELECT status FROM invitations WHERE id = ?", id).Scan(&status)
	return status, err
}

// AcceptTx runs the accept-invitation business logic inside a transaction.
// It returns (coachID, studentID, senderID, err).
func (r *invitationRepository) AcceptTx(invitationID, userID int64) (coachID, studentID, senderID int64, err error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, 0, 0, err
	}
	defer tx.Rollback()

	// Lock and fetch invitation
	var invType string
	var sID, rID int64
	err = tx.QueryRow(
		"SELECT type, sender_id, receiver_id FROM invitations WHERE id = ? AND status = 'pending' FOR UPDATE",
		invitationID,
	).Scan(&invType, &sID, &rID)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invitation not found or already resolved")
	}

	senderID = sID

	// Determine coach and student
	if invType == "coach_invite" {
		coachID = sID
		studentID = rID
	} else {
		// student_request: receiver is coach, sender is student
		coachID = rID
		studentID = sID
	}

	// Check MaxCoachesPerStudent with FOR UPDATE lock
	var activeCount int
	if err = tx.QueryRow(
		"SELECT COUNT(*) FROM coach_students WHERE student_id = ? AND status = 'active' FOR UPDATE",
		studentID,
	).Scan(&activeCount); err != nil {
		return 0, 0, 0, err
	}
	if activeCount >= models.MaxCoachesPerStudent {
		return 0, 0, 0, ErrMaxCoachesReached
	}

	// Create coach_students record
	_, err = tx.Exec(`
		INSERT INTO coach_students (coach_id, student_id, invitation_id, status, started_at)
		VALUES (?, ?, ?, 'active', NOW())
	`, coachID, studentID, invitationID)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to create relationship")
	}

	// Update invitation status
	_, err = tx.Exec("UPDATE invitations SET status = 'accepted', updated_at = NOW() WHERE id = ?", invitationID)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to update invitation")
	}

	if err = tx.Commit(); err != nil {
		return 0, 0, 0, err
	}

	return coachID, studentID, senderID, nil
}

func (r *invitationRepository) Reject(invitationID int64) (senderID int64, err error) {
	_, err = r.db.Exec("UPDATE invitations SET status = 'rejected', updated_at = NOW() WHERE id = ?", invitationID)
	if err != nil {
		return 0, err
	}
	err = r.db.QueryRow("SELECT sender_id FROM invitations WHERE id = ?", invitationID).Scan(&senderID)
	return senderID, err
}

func (r *invitationRepository) FindReceiverByID(receiverID int64) (bool, bool, error) {
	var isCoach, coachPublic bool
	err := r.db.QueryRow(
		"SELECT COALESCE(is_coach, FALSE), COALESCE(coach_public, FALSE) FROM users WHERE id = ?",
		receiverID,
	).Scan(&isCoach, &coachPublic)
	return isCoach, coachPublic, err
}

func (r *invitationRepository) FindReceiverByEmail(email string) (int64, bool, bool, error) {
	var receiverID int64
	var isCoach, coachPublic bool
	err := r.db.QueryRow(
		"SELECT id, COALESCE(is_coach, FALSE), COALESCE(coach_public, FALSE) FROM users WHERE email = ?",
		email,
	).Scan(&receiverID, &isCoach, &coachPublic)
	return receiverID, isCoach, coachPublic, err
}

func (r *invitationRepository) IsSenderCoach(senderID int64) (bool, error) {
	var isCoach bool
	err := r.db.QueryRow("SELECT COALESCE(is_coach, FALSE) FROM users WHERE id = ?", senderID).Scan(&isCoach)
	return isCoach, err
}

func (r *invitationRepository) CountPending(userID, otherID int64) (int, error) {
	var count int
	err := r.db.QueryRow(`
        SELECT COUNT(*) FROM invitations WHERE status = 'pending' AND (
            (sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)
        )
    `, userID, otherID, otherID, userID).Scan(&count)
	return count, err
}

func (r *invitationRepository) CountActiveRelationship(userID, otherID int64) (int, error) {
	var count int
	err := r.db.QueryRow(`
        SELECT COUNT(*) FROM coach_students WHERE status = 'active' AND (
            (coach_id = ? AND student_id = ?) OR (coach_id = ? AND student_id = ?)
        )
    `, userID, otherID, otherID, userID).Scan(&count)
	return count, err
}

func (r *invitationRepository) CountStudentActiveCoaches(studentID int64) (int, error) {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM coach_students WHERE student_id = ? AND status = 'active'",
		studentID,
	).Scan(&count)
	return count, err
}

func (r *invitationRepository) Create(senderID, receiverID int64, invType, message string) (int64, error) {
	result, err := r.db.Exec(
		"INSERT INTO invitations (type, sender_id, receiver_id, message, status) VALUES (?, ?, ?, ?, 'pending')",
		invType, senderID, receiverID, message,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *invitationRepository) GetByID(id int64) (models.Invitation, error) {
	var inv models.Invitation
	err := r.db.QueryRow(`
        SELECT i.id, i.type, i.sender_id, COALESCE(i.receiver_id, 0), COALESCE(i.receiver_email, ''), COALESCE(i.message, ''), i.status, i.created_at, i.updated_at,
            COALESCE(s.name, ''), COALESCE(s.custom_avatar, ''), COALESCE(rv.name, ''), COALESCE(rv.custom_avatar, '')
        FROM invitations i
        JOIN users s ON s.id = i.sender_id
        LEFT JOIN users rv ON rv.id = i.receiver_id
        WHERE i.id = ?
    `, id).Scan(&inv.ID, &inv.Type, &inv.SenderID, &inv.ReceiverID, &inv.ReceiverEmail, &inv.Message, &inv.Status, &inv.CreatedAt, &inv.UpdatedAt,
		&inv.SenderName, &inv.SenderAvatar, &inv.ReceiverName, &inv.ReceiverAvatar)
	return inv, err
}

func (r *invitationRepository) List(userID int64, status, direction string, limit, offset int) ([]models.Invitation, error) {
	query := `
        SELECT i.id, i.type, i.sender_id, COALESCE(i.receiver_id, 0), COALESCE(i.receiver_email, ''), COALESCE(i.message, ''), i.status, i.created_at, i.updated_at,
            COALESCE(s.name, ''), COALESCE(s.custom_avatar, ''), COALESCE(rv.name, ''), COALESCE(rv.custom_avatar, '')
        FROM invitations i
        JOIN users s ON s.id = i.sender_id
        LEFT JOIN users rv ON rv.id = i.receiver_id
        WHERE 1=1
    `
	args := []interface{}{}

	switch direction {
	case "sent":
		query += " AND i.sender_id = ?"
		args = append(args, userID)
	case "received":
		query += " AND i.receiver_id = ?"
		args = append(args, userID)
	default:
		query += " AND (i.sender_id = ? OR i.receiver_id = ?)"
		args = append(args, userID, userID)
	}
	if status != "" {
		query += " AND i.status = ?"
		args = append(args, status)
	}
	query += " ORDER BY i.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	invitations := []models.Invitation{}
	for rows.Next() {
		var inv models.Invitation
		if err := rows.Scan(&inv.ID, &inv.Type, &inv.SenderID, &inv.ReceiverID, &inv.ReceiverEmail, &inv.Message, &inv.Status, &inv.CreatedAt, &inv.UpdatedAt,
			&inv.SenderName, &inv.SenderAvatar, &inv.ReceiverName, &inv.ReceiverAvatar); err != nil {
			continue
		}
		invitations = append(invitations, inv)
	}
	return invitations, nil
}

func (r *invitationRepository) Cancel(invID int64) error {
	_, err := r.db.Exec("UPDATE invitations SET status = 'cancelled', updated_at = NOW() WHERE id = ?", invID)
	return err
}

func (r *invitationRepository) IsAdmin(userID int64) (bool, error) {
	var isAdmin bool
	err := r.db.QueryRow("SELECT COALESCE(is_admin, FALSE) FROM users WHERE id = ?", userID).Scan(&isAdmin)
	return isAdmin, err
}

func (r *invitationRepository) CreateForUnknown(senderID int64, invType, message, receiverEmail, inviteToken string) (int64, error) {
	result, err := r.db.Exec(
		"INSERT INTO invitations (type, sender_id, receiver_id, receiver_email, invite_token, message, status) VALUES (?, ?, NULL, ?, ?, ?, 'pending')",
		invType, senderID, receiverEmail, inviteToken, message,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *invitationRepository) FindByToken(token string) (models.Invitation, error) {
	var inv models.Invitation
	err := r.db.QueryRow(`
        SELECT i.id, i.type, i.sender_id, COALESCE(i.receiver_id, 0), COALESCE(i.receiver_email, ''), COALESCE(i.message, ''), i.status, i.created_at, i.updated_at,
            COALESCE(s.name, ''), COALESCE(s.custom_avatar, ''), '', ''
        FROM invitations i
        JOIN users s ON s.id = i.sender_id
        WHERE i.invite_token = ? AND i.status = 'pending'
    `, token).Scan(&inv.ID, &inv.Type, &inv.SenderID, &inv.ReceiverID, &inv.ReceiverEmail, &inv.Message, &inv.Status, &inv.CreatedAt, &inv.UpdatedAt,
		&inv.SenderName, &inv.SenderAvatar, &inv.ReceiverName, &inv.ReceiverAvatar)
	return inv, err
}

func (r *invitationRepository) RedeemToken(token string, userID int64) error {
	result, err := r.db.Exec(
		"UPDATE invitations SET receiver_id = ?, receiver_email = NULL, invite_token = NULL, updated_at = NOW() WHERE invite_token = ? AND status = 'pending' AND receiver_id IS NULL",
		userID, token,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *invitationRepository) FindPendingByEmail(email string) ([]models.Invitation, error) {
	rows, err := r.db.Query(
		"SELECT id FROM invitations WHERE receiver_email = ? AND status = 'pending'",
		email,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invitations []models.Invitation
	for rows.Next() {
		var inv models.Invitation
		if err := rows.Scan(&inv.ID); err != nil {
			continue
		}
		invitations = append(invitations, inv)
	}
	return invitations, nil
}
