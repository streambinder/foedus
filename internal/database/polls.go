package database

import (
	"fmt"
	"strings"

	"github.com/streambinder/foedus/internal/models"
)

func CreatePoll(question, description string) error {
	_, err := DB.Exec(`INSERT INTO polls (question, description) VALUES (?, ?)`, question, description)
	return err
}

func UpdatePoll(id int, question, description string) error {
	_, err := DB.Exec(`UPDATE polls SET question = ?, description = ? WHERE id = ?`, question, description, id)
	return err
}

func GetPoll(id int) (models.Poll, error) {
	var p models.Poll
	err := DB.QueryRow(`SELECT id, question, description, created_at FROM polls WHERE id = ?`, id).Scan(&p.ID, &p.Question, &p.Description, &p.CreatedAt)
	return p, err
}

func DeletePoll(id int) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM poll_answers WHERE poll_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM polls WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func GetAllPollsWithCounts() ([]models.Poll, error) {
	rows, err := DB.Query(`
		SELECT p.id, p.question, p.description, p.created_at, COUNT(pa.id)
		FROM polls p
		LEFT JOIN poll_answers pa ON pa.poll_id = p.id
		GROUP BY p.id
		ORDER BY p.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var polls []models.Poll
	for rows.Next() {
		var p models.Poll
		if err := rows.Scan(&p.ID, &p.Question, &p.Description, &p.CreatedAt, &p.TotalCount); err != nil {
			return nil, err
		}
		polls = append(polls, p)
	}

	// load yes-voter names per poll
	for i := range polls {
		voterRows, err := DB.Query(
			`SELECT g.first_name || ' ' || g.last_name FROM poll_answers pa JOIN guests g ON g.id = pa.guest_id WHERE pa.poll_id = ? AND pa.answer = 1 ORDER BY g.first_name`,
			polls[i].ID,
		)
		if err != nil {
			return nil, err
		}
		for voterRows.Next() {
			var name string
			if err := voterRows.Scan(&name); err != nil {
				voterRows.Close()
				return nil, err
			}
			polls[i].YesVoters = append(polls[i].YesVoters, name)
		}
		voterRows.Close()
	}

	return polls, nil
}

func GetAllPolls() ([]models.Poll, error) {
	rows, err := DB.Query(`SELECT id, question, description, created_at FROM polls ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var polls []models.Poll
	for rows.Next() {
		var p models.Poll
		if err := rows.Scan(&p.ID, &p.Question, &p.Description, &p.CreatedAt); err != nil {
			return nil, err
		}
		polls = append(polls, p)
	}
	return polls, nil
}

func SavePollAnswers(guestID int, answers map[int]models.PollAnswer) error {
	if len(answers) == 0 {
		return nil
	}

	// build a single INSERT OR REPLACE statement
	var placeholders []string
	var args []any
	for pollID, answer := range answers {
		placeholders = append(placeholders, "(?, ?, ?, ?)")
		answerInt := 0
		if answer.Answer {
			answerInt = 1
		}
		args = append(args, pollID, guestID, answerInt, answer.Notes)
	}
	_, err := DB.Exec(
		fmt.Sprintf(`INSERT OR REPLACE INTO poll_answers (poll_id, guest_id, answer, notes) VALUES %s`, strings.Join(placeholders, ",")),
		args...,
	)
	return err
}

func GetPollAnswersForGuests(guestIDs []int) (map[int][]models.PollAnswer, error) {
	if len(guestIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(guestIDs))
	args := make([]any, len(guestIDs))
	for i, id := range guestIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	rows, err := DB.Query(
		fmt.Sprintf(`SELECT guest_id, poll_id, answer, notes FROM poll_answers WHERE guest_id IN (%s)`, strings.Join(placeholders, ",")),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int][]models.PollAnswer)
	for rows.Next() {
		var guestID, pollID, answer int
		var notes string
		if err := rows.Scan(&guestID, &pollID, &answer, &notes); err != nil {
			return nil, err
		}
		result[guestID] = append(result[guestID], models.PollAnswer{PollID: pollID, Answer: answer == 1, Notes: notes})
	}
	return result, nil
}
