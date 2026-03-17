package repository

import (
	"database/sql"
	"fmt"

	"github.com/fitreg/api/models"
)

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
		return 0, 0, 0, fmt.Errorf("student has reached the maximum number of coaches (%d)", models.MaxCoachesPerStudent)
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
