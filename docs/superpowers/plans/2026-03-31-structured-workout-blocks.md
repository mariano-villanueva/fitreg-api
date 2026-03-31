# Structured Workout Blocks — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `rest` and `block` segment types so coaches can express structured workouts like "3 blocks of 1min fast / 1min rest / 2min fast / 1min rest / 1min fast" without manually duplicating rows.

**Architecture:** Additive DB migration (new `parent_id` column + ENUM extension), two-pass repository insert for blocks, flat-array API with tree construction on the frontend. Blocks are 1 level deep only.

**Tech Stack:** Go (stdlib, MySQL), React 19 + TypeScript, react-i18next

---

## File Map

### Backend (`FitRegAPI/`)

| File | Change |
|------|--------|
| `migrations/001_schema.sql` | Add migration block for `parent_id` + ENUM extension on 3 tables |
| `models/coach.go` | Add `ParentID *int64` + `TempID *int64` to `WorkoutSegment` and `SegmentRequest` |
| `repository/workout_repository.go` | Update `GetSegments` (select parent_id); rewrite `ReplaceSegments` (two-pass) |
| `repository/template_repository.go` | Same segment changes for `workout_template_segments` |
| `repository/weekly_template_repository.go` | Same segment changes for `weekly_template_day_segments` |
| `handlers/workout_handler_test.go` | Add tests for create/update with block+rest segments |

### Frontend (`FitRegFE/src/`)

| File | Change |
|------|--------|
| `types/index.ts` | Extend `WorkoutSegment` (parent_id, new segment_type values, intensity as string) |
| `i18n/es.ts` | Add keys: `segment_rest`, `segment_block`, `segment_add_rest`, `segment_add_block`, `segment_block_reps` |
| `i18n/en.ts` | Same keys in English |
| `components/SegmentBuilder.tsx` | Rewrite: tree state, block inline rendering, rest type support |
| `components/SegmentDisplay.tsx` | Add block group rendering + rest display |

---

## Task 1: DB Migration

**Files:**
- Modify: `migrations/001_schema.sql`

- [ ] **Step 1: Append the migration block to the schema file**

Open `migrations/001_schema.sql` and append at the end:

```sql
-- Migration: structured workout blocks
-- Adds parent_id (for block children) and extends segment_type ENUM

ALTER TABLE workout_segments
  ADD COLUMN parent_id BIGINT NULL,
  ADD CONSTRAINT fk_ws_parent FOREIGN KEY (parent_id) REFERENCES workout_segments(id) ON DELETE CASCADE,
  MODIFY COLUMN segment_type ENUM('simple','interval','rest','block') NOT NULL DEFAULT 'simple';

ALTER TABLE workout_template_segments
  ADD COLUMN parent_id BIGINT NULL,
  ADD CONSTRAINT fk_wts_parent FOREIGN KEY (parent_id) REFERENCES workout_template_segments(id) ON DELETE CASCADE,
  MODIFY COLUMN segment_type ENUM('simple','interval','rest','block') NOT NULL DEFAULT 'simple';

ALTER TABLE weekly_template_day_segments
  ADD COLUMN parent_id BIGINT NULL,
  ADD CONSTRAINT fk_wtds_parent FOREIGN KEY (parent_id) REFERENCES weekly_template_day_segments(id) ON DELETE CASCADE,
  MODIFY COLUMN segment_type ENUM('simple','interval','rest','block') NOT NULL DEFAULT 'simple';
```

- [ ] **Step 2: Apply migration to local DB**

```bash
cd ~/Desktop/FitReg/FitRegAPI
mysql -u root fitreg < migrations/001_schema.sql
```

Expected: no errors. If you get "Duplicate column name 'parent_id'", the migration was already applied — skip.

- [ ] **Step 3: Verify schema**

```bash
mysql -u root fitreg -e "DESCRIBE workout_segments;"
```

Expected output includes: `parent_id  bigint  YES  MUL  NULL` and `segment_type` shows `enum('simple','interval','rest','block')`.

- [ ] **Step 4: Commit**

```bash
cd ~/Desktop/FitReg/FitRegAPI
git add migrations/001_schema.sql
git commit -m "feat: add parent_id and rest/block segment types to schema"
```

---

## Task 2: Go Model — Add ParentID and TempID

**Files:**
- Modify: `models/coach.go`

Find `WorkoutSegment` struct and `SegmentRequest` struct. Add `ParentID` and `TempID` fields to both.

- [ ] **Step 1: Update WorkoutSegment**

In `models/coach.go`, find the `WorkoutSegment` struct (search for `type WorkoutSegment struct`). It currently starts with:
```go
type WorkoutSegment struct {
    ID            int64   `json:"id"`
    WorkoutID     int64   `json:"workout_id,omitempty"`
    OrderIndex    int     `json:"order_index"`
    SegmentType   string  `json:"segment_type"`
```

Add `ParentID *int64` after `ID`:

```go
type WorkoutSegment struct {
    ID            int64   `json:"id"`
    ParentID      *int64  `json:"parent_id"`
    WorkoutID     int64   `json:"workout_id,omitempty"`
    OrderIndex    int     `json:"order_index"`
    SegmentType   string  `json:"segment_type"`
    Repetitions   int     `json:"repetitions"`
    Value         float64 `json:"value"`
    Unit          string  `json:"unit"`
    Intensity     string  `json:"intensity"`
    WorkValue     float64 `json:"work_value"`
    WorkUnit      string  `json:"work_unit"`
    WorkIntensity string  `json:"work_intensity"`
    RestValue     float64 `json:"rest_value"`
    RestUnit      string  `json:"rest_unit"`
    RestIntensity string  `json:"rest_intensity"`
}
```

- [ ] **Step 2: Update SegmentRequest**

Find `type SegmentRequest struct` (it may be in `models/coach.go` or a separate file — run `grep -r "type SegmentRequest" ~/Desktop/FitReg/FitRegAPI/`). Add `TempID` and `ParentID`:

```go
type SegmentRequest struct {
    TempID        *int64  `json:"temp_id"`    // client-assigned negative ID for block segments; nil otherwise
    ParentID      *int64  `json:"parent_id"`  // references TempID of parent block; nil for root segments
    OrderIndex    int     `json:"order_index"`
    SegmentType   string  `json:"segment_type"`
    Repetitions   int     `json:"repetitions"`
    Value         float64 `json:"value"`
    Unit          string  `json:"unit"`
    Intensity     string  `json:"intensity"`
    WorkValue     float64 `json:"work_value"`
    WorkUnit      string  `json:"work_unit"`
    WorkIntensity string  `json:"work_intensity"`
    RestValue     float64 `json:"rest_value"`
    RestUnit      string  `json:"rest_unit"`
    RestIntensity string  `json:"rest_intensity"`
}
```

- [ ] **Step 3: Verify the project still compiles**

```bash
cd ~/Desktop/FitReg/FitRegAPI
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add models/
git commit -m "feat: add ParentID and TempID to WorkoutSegment and SegmentRequest"
```

---

## Task 3: Repository — GetSegments

**Files:**
- Modify: `repository/workout_repository.go`

- [ ] **Step 1: Write a failing test**

In `handlers/workout_handler_test.go`, verify that `GetWorkout` returns segments with `parent_id`. Add after existing tests:

```go
func TestWorkoutHandler_GetWorkout_IncludesParentID(t *testing.T) {
    blockParentID := int64(10)
    mock := &mockWorkoutService{
        getByIDFn: func(id, userID int64) (models.Workout, error) {
            return models.Workout{
                ID:     id,
                UserID: userID,
                Segments: []models.WorkoutSegment{
                    {ID: 10, ParentID: nil,            OrderIndex: 0, SegmentType: "block", Repetitions: 3},
                    {ID: 11, ParentID: &blockParentID, OrderIndex: 0, SegmentType: "simple", Value: 1, Unit: "min", Intensity: "fast"},
                    {ID: 12, ParentID: &blockParentID, OrderIndex: 1, SegmentType: "rest",   Value: 1, Unit: "min"},
                },
            }, nil
        },
    }
    h := NewWorkoutHandler(mock)
    w := httptest.NewRecorder()
    h.GetWorkout(w, newWorkoutReq(http.MethodGet, "/api/workouts/1", nil, 42))

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", w.Code)
    }
    var resp models.Workout
    if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
        t.Fatal(err)
    }
    if len(resp.Segments) != 3 {
        t.Fatalf("expected 3 segments, got %d", len(resp.Segments))
    }
    if resp.Segments[0].ParentID != nil {
        t.Error("expected block segment to have nil parent_id")
    }
    if resp.Segments[1].ParentID == nil || *resp.Segments[1].ParentID != 10 {
        t.Error("expected child segment parent_id = 10")
    }
    if resp.Segments[2].SegmentType != "rest" {
        t.Errorf("expected rest segment, got %s", resp.Segments[2].SegmentType)
    }
}
```

- [ ] **Step 2: Run test — expect compile error on new model fields**

```bash
cd ~/Desktop/FitReg/FitRegAPI
go test ./handlers/... -run TestWorkoutHandler_GetWorkout_IncludesParentID -v
```

Expected: compiles and passes (the handler just serialises what the service returns — this tests the JSON shape, which works once Task 2 is done). If it passes green, continue to the repository change.

- [ ] **Step 3: Update GetSegments to select parent_id**

In `repository/workout_repository.go`, find `GetSegments`. Replace:

```go
func (r *workoutRepository) GetSegments(workoutID int64) ([]models.WorkoutSegment, error) {
	rows, err := r.db.Query(`
		SELECT id, workout_id, order_index, segment_type, COALESCE(repetitions, 1),
			COALESCE(value, 0), COALESCE(unit, ''), COALESCE(intensity, ''),
			COALESCE(work_value, 0), COALESCE(work_unit, ''), COALESCE(work_intensity, ''),
			COALESCE(rest_value, 0), COALESCE(rest_unit, ''), COALESCE(rest_intensity, '')
		FROM workout_segments WHERE workout_id = ? ORDER BY order_index
	`, workoutID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	segments := []models.WorkoutSegment{}
	for rows.Next() {
		var s models.WorkoutSegment
		if err := rows.Scan(&s.ID, &s.WorkoutID, &s.OrderIndex, &s.SegmentType, &s.Repetitions,
			&s.Value, &s.Unit, &s.Intensity,
			&s.WorkValue, &s.WorkUnit, &s.WorkIntensity,
			&s.RestValue, &s.RestUnit, &s.RestIntensity); err != nil {
			return nil, err
		}
		segments = append(segments, s)
	}
	return segments, rows.Err()
}
```

With:

```go
func (r *workoutRepository) GetSegments(workoutID int64) ([]models.WorkoutSegment, error) {
	rows, err := r.db.Query(`
		SELECT id, parent_id, workout_id, order_index, segment_type, COALESCE(repetitions, 1),
			COALESCE(value, 0), COALESCE(unit, ''), COALESCE(intensity, ''),
			COALESCE(work_value, 0), COALESCE(work_unit, ''), COALESCE(work_intensity, ''),
			COALESCE(rest_value, 0), COALESCE(rest_unit, ''), COALESCE(rest_intensity, '')
		FROM workout_segments WHERE workout_id = ? ORDER BY order_index
	`, workoutID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	segments := []models.WorkoutSegment{}
	for rows.Next() {
		var s models.WorkoutSegment
		var parentID sql.NullInt64
		if err := rows.Scan(&s.ID, &parentID, &s.WorkoutID, &s.OrderIndex, &s.SegmentType, &s.Repetitions,
			&s.Value, &s.Unit, &s.Intensity,
			&s.WorkValue, &s.WorkUnit, &s.WorkIntensity,
			&s.RestValue, &s.RestUnit, &s.RestIntensity); err != nil {
			return nil, err
		}
		if parentID.Valid {
			v := parentID.Int64
			s.ParentID = &v
		}
		segments = append(segments, s)
	}
	return segments, rows.Err()
}
```

- [ ] **Step 4: Build and run test**

```bash
go build ./... && go test ./handlers/... -run TestWorkoutHandler_GetWorkout_IncludesParentID -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add repository/workout_repository.go handlers/workout_handler_test.go
git commit -m "feat: GetSegments includes parent_id in response"
```

---

## Task 4: Repository — ReplaceSegments (two-pass insert)

**Files:**
- Modify: `repository/workout_repository.go`

- [ ] **Step 1: Write a failing handler test for create with blocks**

Add to `handlers/workout_handler_test.go`:

```go
func TestWorkoutHandler_Create_WithBlockSegments(t *testing.T) {
    tempBlockID := int64(-1)
    reqSegments := []models.SegmentRequest{
        // root simple
        {OrderIndex: 0, SegmentType: "simple", Value: 15, Unit: "min", Intensity: "easy"},
        // block with TempID = -1
        {TempID: &tempBlockID, OrderIndex: 1, SegmentType: "block", Repetitions: 3},
        // children of block
        {ParentID: &tempBlockID, OrderIndex: 0, SegmentType: "simple", Value: 1, Unit: "min", Intensity: "fast"},
        {ParentID: &tempBlockID, OrderIndex: 1, SegmentType: "rest", Value: 1, Unit: "min"},
    }

    var capturedSegments []models.SegmentRequest
    mock := &mockWorkoutService{
        createFn: func(userID int64, req models.CreateWorkoutRequest) (models.Workout, error) {
            capturedSegments = req.Segments
            return models.Workout{ID: 1, UserID: userID, DueDate: req.DueDate}, nil
        },
    }
    h := NewWorkoutHandler(mock)

    body, _ := json.Marshal(models.CreateWorkoutRequest{
        DueDate:  "2026-04-01",
        Type:     "fartlek",
        Segments: reqSegments,
    })
    w := httptest.NewRecorder()
    h.CreateWorkout(w, newWorkoutReq(http.MethodPost, "/api/workouts", body, 42))

    if w.Code != http.StatusCreated {
        t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
    }
    if len(capturedSegments) != 4 {
        t.Fatalf("expected 4 segments passed to service, got %d", len(capturedSegments))
    }
    // verify block has TempID set
    if capturedSegments[1].TempID == nil || *capturedSegments[1].TempID != -1 {
        t.Error("expected block segment TempID = -1")
    }
    // verify child references block
    if capturedSegments[2].ParentID == nil || *capturedSegments[2].ParentID != -1 {
        t.Error("expected child segment ParentID = -1")
    }
    // verify rest type
    if capturedSegments[3].SegmentType != "rest" {
        t.Errorf("expected rest segment, got %s", capturedSegments[3].SegmentType)
    }
}
```

- [ ] **Step 2: Run test — should pass (handler just passes segments through)**

```bash
go test ./handlers/... -run TestWorkoutHandler_Create_WithBlockSegments -v
```

Expected: PASS. The handler doesn't transform segments, it just decodes and passes them to the service.

- [ ] **Step 3: Rewrite ReplaceSegments with two-pass insert**

In `repository/workout_repository.go`, replace the entire `ReplaceSegments` method:

```go
func (r *workoutRepository) ReplaceSegments(workoutID int64, segs []models.SegmentRequest) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM workout_segments WHERE workout_id = ?", workoutID); err != nil {
		return err
	}

	// tempIDToRealID maps client-assigned negative TempID values to DB-generated IDs.
	tempIDToRealID := map[int64]int64{}

	// Pass 1: insert root-level segments (ParentID is nil).
	for i, seg := range segs {
		if seg.ParentID != nil {
			continue // child segment, handled in pass 2
		}
		result, err := tx.Exec(`
			INSERT INTO workout_segments
			  (workout_id, parent_id, order_index, segment_type, repetitions, value, unit, intensity,
			   work_value, work_unit, work_intensity, rest_value, rest_unit, rest_intensity)
			VALUES (?, NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, workoutID, i, seg.SegmentType, seg.Repetitions, seg.Value, seg.Unit, seg.Intensity,
			seg.WorkValue, seg.WorkUnit, seg.WorkIntensity, seg.RestValue, seg.RestUnit, seg.RestIntensity)
		if err != nil {
			return err
		}
		if seg.SegmentType == "block" && seg.TempID != nil {
			realID, err := result.LastInsertId()
			if err != nil {
				return err
			}
			tempIDToRealID[*seg.TempID] = realID
		}
	}

	// Pass 2: insert child segments (ParentID is not nil).
	for _, seg := range segs {
		if seg.ParentID == nil {
			continue
		}
		realParentID, ok := tempIDToRealID[*seg.ParentID]
		if !ok {
			return fmt.Errorf("segment references unknown block temp_id %d", *seg.ParentID)
		}
		if _, err := tx.Exec(`
			INSERT INTO workout_segments
			  (workout_id, parent_id, order_index, segment_type, repetitions, value, unit, intensity,
			   work_value, work_unit, work_intensity, rest_value, rest_unit, rest_intensity)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, workoutID, realParentID, seg.OrderIndex, seg.SegmentType, seg.Repetitions,
			seg.Value, seg.Unit, seg.Intensity,
			seg.WorkValue, seg.WorkUnit, seg.WorkIntensity,
			seg.RestValue, seg.RestUnit, seg.RestIntensity); err != nil {
			return err
		}
	}

	return tx.Commit()
}
```

Add `"fmt"` to the import block of `workout_repository.go` if not already present.

- [ ] **Step 4: Build**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 5: Run all handler tests**

```bash
go test ./handlers/... -v 2>&1 | tail -20
```

Expected: all existing tests pass.

- [ ] **Step 6: Commit**

```bash
git add repository/workout_repository.go handlers/workout_handler_test.go
git commit -m "feat: ReplaceSegments supports block/rest via two-pass insert"
```

---

## Task 5: Template Repositories — Segment Changes

**Files:**
- Modify: `repository/template_repository.go` (for `workout_template_segments`)
- Modify: `repository/weekly_template_repository.go` (for `weekly_template_day_segments`)

The pattern is identical to Task 3+4 for each repo's `GetSegments` / `ReplaceSegments` equivalents.

- [ ] **Step 1: Find the segment methods in template repos**

```bash
grep -n "GetSegments\|ReplaceSegments\|INSERT INTO workout_template_segments\|INSERT INTO weekly_template_day_segments" \
  ~/Desktop/FitReg/FitRegAPI/repository/template_repository.go \
  ~/Desktop/FitReg/FitRegAPI/repository/weekly_template_repository.go
```

Note the line numbers returned.

- [ ] **Step 2: Update workout_template_segments read**

Find the `SELECT` query that reads from `workout_template_segments`. Add `parent_id` to the SELECT columns and scan it into `sql.NullInt64`, same pattern as Task 3 Step 3. The column list goes from:

```sql
SELECT id, template_id, order_index, segment_type, ...
```

To:

```sql
SELECT id, parent_id, template_id, order_index, segment_type, ...
```

And in the scan loop, add:

```go
var parentID sql.NullInt64
// include &parentID in Scan call
if parentID.Valid {
    v := parentID.Int64
    s.ParentID = &v
}
```

- [ ] **Step 3: Update workout_template_segments write**

Find the INSERT in `ReplaceSegments` for templates. Apply the two-pass logic from Task 4 Step 3, substituting the table name `workout_template_segments` and the FK column (likely `template_id` instead of `workout_id`).

- [ ] **Step 4: Repeat for weekly_template_day_segments**

Same two changes (read + write) for the weekly template segment table. The FK column is likely `weekly_template_day_id`.

- [ ] **Step 5: Build**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 6: Run all tests**

```bash
go test ./... 2>&1 | tail -10
```

Expected: no failures.

- [ ] **Step 7: Commit**

```bash
git add repository/
git commit -m "feat: template segment repos support parent_id for block/rest types"
```

---

## Task 6: Frontend — TypeScript Types

**Files:**
- Modify: `FitRegFE/src/types/index.ts`

- [ ] **Step 1: Extend WorkoutSegment**

Find the `WorkoutSegment` interface. Replace it with:

```typescript
export interface WorkoutSegment {
  id?: number;
  parent_id?: number | null;
  workout_id?: number;
  order_index: number;
  segment_type: 'simple' | 'interval' | 'rest' | 'block';
  repetitions: number;
  value: number;
  unit: string;
  intensity: string;
  work_value: number;
  work_unit: string;
  work_intensity: string;
  rest_value: number;
  rest_unit: string;
  rest_intensity: string;
  // temp_id is assigned by the frontend to blocks before API submission
  temp_id?: number | null;
}
```

Note: `unit`, `intensity`, `work_unit`, `work_intensity`, `rest_unit`, `rest_intensity` are all widened to `string`. The union types are no longer needed since the selects are driven by the `UNITS` and `INTENSITIES` constant arrays in SegmentBuilder, and `rest` uses free-text intensity. The runtime values are unchanged.

- [ ] **Step 2: Add SegmentNode type (tree representation)**

Below `WorkoutSegment`, add:

```typescript
// SegmentNode is the internal tree representation used by SegmentBuilder.
// block nodes have children; all others have children = [].
export interface SegmentNode extends WorkoutSegment {
  children: SegmentNode[];
}
```

- [ ] **Step 3: Verify no TypeScript errors**

```bash
cd ~/Desktop/FitReg/FitRegFE
npm run build 2>&1 | grep -E "error TS|Error"
```

Expected: no new type errors. If existing components reference `intensity` as the old union type in a way that breaks, change those references to `string` — the underlying values ('easy', 'moderate', 'fast', 'sprint') are unchanged.

- [ ] **Step 4: Commit**

```bash
cd ~/Desktop/FitReg/FitRegFE
git add src/types/index.ts
git commit -m "feat: extend WorkoutSegment with parent_id, block/rest types, SegmentNode"
```

---

## Task 7: i18n Keys

**Files:**
- Modify: `FitRegFE/src/i18n/es.ts`
- Modify: `FitRegFE/src/i18n/en.ts`

- [ ] **Step 1: Add Spanish keys**

In `es.ts`, find the `segment_*` key group and add:

```typescript
segment_rest: 'Descanso',
segment_block: 'Bloque',
segment_add_rest: 'Descanso',
segment_add_block: 'Bloque',
segment_block_reps: '× {{count}} veces',
segment_block_children: '{{count}} segmentos',
```

- [ ] **Step 2: Add English keys**

In `en.ts`, same location:

```typescript
segment_rest: 'Rest',
segment_block: 'Block',
segment_add_rest: 'Rest',
segment_add_block: 'Block',
segment_block_reps: '× {{count}} times',
segment_block_children: '{{count}} segments',
```

- [ ] **Step 3: Commit**

```bash
git add src/i18n/
git commit -m "feat: add i18n keys for rest and block segment types"
```

---

## Task 8: SegmentBuilder Rewrite

**Files:**
- Modify: `FitRegFE/src/components/SegmentBuilder.tsx`

This is the largest change. The component is rewritten to work with a tree internally (`SegmentNode[]`) and expose a flat `WorkoutSegment[]` to the parent via `onChange`.

- [ ] **Step 1: Add helper functions (tree ↔ flat conversion) and factory functions**

Replace the entire file content with the following. Read it completely before applying.

```tsx
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { WorkoutSegment, SegmentNode } from '../types';

interface SegmentBuilderProps {
  segments: WorkoutSegment[];
  onChange: (segments: WorkoutSegment[]) => void;
}

const UNITS = ['km', 'm', 'min', 'sec'] as const;
const INTENSITIES = ['easy', 'moderate', 'fast', 'sprint'] as const;

// ─── Factory functions ────────────────────────────────────────────────────────

function makeSimple(): SegmentNode {
  return {
    segment_type: 'simple', order_index: 0, repetitions: 1,
    value: 1, unit: 'km', intensity: 'easy',
    work_value: 0, work_unit: 'km', work_intensity: 'fast',
    rest_value: 0, rest_unit: 'km', rest_intensity: 'easy',
    children: [],
  };
}

function makeInterval(): SegmentNode {
  return {
    segment_type: 'interval', order_index: 0, repetitions: 3,
    value: 0, unit: 'km', intensity: 'easy',
    work_value: 1, work_unit: 'min', work_intensity: 'fast',
    rest_value: 1, rest_unit: 'min', rest_intensity: 'easy',
    children: [],
  };
}

function makeRest(): SegmentNode {
  return {
    segment_type: 'rest', order_index: 0, repetitions: 1,
    value: 1, unit: 'min', intensity: '',
    work_value: 0, work_unit: 'min', work_intensity: '',
    rest_value: 0, rest_unit: 'min', rest_intensity: '',
    children: [],
  };
}

function makeBlock(): SegmentNode {
  return {
    segment_type: 'block', order_index: 0, repetitions: 3,
    value: 0, unit: 'min', intensity: '',
    work_value: 0, work_unit: 'min', work_intensity: '',
    rest_value: 0, rest_unit: 'min', rest_intensity: '',
    children: [],
  };
}

// ─── Tree ↔ Flat conversion ───────────────────────────────────────────────────

function buildTree(flat: WorkoutSegment[]): SegmentNode[] {
  const roots = flat
    .filter(s => !s.parent_id)
    .sort((a, b) => a.order_index - b.order_index)
    .map(s => ({ ...s, children: [] as SegmentNode[] }));

  for (const root of roots) {
    if (root.segment_type === 'block' && root.id) {
      root.children = flat
        .filter(s => s.parent_id === root.id)
        .sort((a, b) => a.order_index - b.order_index)
        .map(s => ({ ...s, children: [] }));
    }
  }
  return roots;
}

function flattenForAPI(nodes: SegmentNode[]): WorkoutSegment[] {
  const flat: WorkoutSegment[] = [];
  let tempBlockCounter = -1;

  nodes.forEach((node, rootIdx) => {
    if (node.segment_type === 'block') {
      const tempId = tempBlockCounter--;
      flat.push({ ...node, order_index: rootIdx, parent_id: null, temp_id: tempId, children: undefined as never });
      node.children.forEach((child, childIdx) => {
        flat.push({ ...child, order_index: childIdx, parent_id: tempId, temp_id: null, children: undefined as never });
      });
    } else {
      flat.push({ ...node, order_index: rootIdx, parent_id: null, children: undefined as never });
    }
  });
  return flat;
}

function reindexNodes(nodes: SegmentNode[]): SegmentNode[] {
  return nodes.map((n, i) => ({ ...n, order_index: i }));
}

// ─── Component ───────────────────────────────────────────────────────────────

export default function SegmentBuilder({ segments, onChange }: SegmentBuilderProps) {
  const { t } = useTranslation();
  const [tree, setTree] = useState<SegmentNode[]>(() => buildTree(segments));

  // editPath: [rootIndex] for root segments, [rootIndex, childIndex] for children
  const [editPath, setEditPath] = useState<number[] | null>(null);
  const [editDraft, setEditDraft] = useState<SegmentNode | null>(null);
  const [menuPath, setMenuPath] = useState<number[] | null>(null);

  function commit(newTree: SegmentNode[]) {
    const indexed = reindexNodes(newTree);
    setTree(indexed);
    onChange(flattenForAPI(indexed));
  }

  // ─── Root-level operations ────────────────────────────────────────────────

  function addRoot(factory: () => SegmentNode) {
    const node = factory();
    const newTree = [...tree, node];
    commit(newTree);
    const newIdx = newTree.length - 1;
    setEditDraft({ ...node });
    setEditPath([newIdx]);
  }

  function removeRoot(idx: number) {
    commit(tree.filter((_, i) => i !== idx));
    setMenuPath(null);
  }

  function moveRootUp(idx: number) {
    if (idx === 0) return;
    const copy = [...tree];
    [copy[idx - 1], copy[idx]] = [copy[idx], copy[idx - 1]];
    commit(copy);
    setMenuPath(null);
  }

  function moveRootDown(idx: number) {
    if (idx === tree.length - 1) return;
    const copy = [...tree];
    [copy[idx], copy[idx + 1]] = [copy[idx + 1], copy[idx]];
    commit(copy);
    setMenuPath(null);
  }

  function duplicateRoot(idx: number) {
    const copy = [...tree];
    copy.splice(idx + 1, 0, { ...tree[idx], children: [...tree[idx].children] });
    commit(copy);
    setMenuPath(null);
  }

  // ─── Child operations (inside a block) ───────────────────────────────────

  function addChild(blockIdx: number, factory: () => SegmentNode) {
    const node = factory();
    const newTree = tree.map((block, i) => {
      if (i !== blockIdx) return block;
      const newChildren = [...block.children, node];
      return { ...block, children: newChildren };
    });
    commit(newTree);
    const childIdx = newTree[blockIdx].children.length - 1;
    setEditDraft({ ...node });
    setEditPath([blockIdx, childIdx]);
  }

  function removeChild(blockIdx: number, childIdx: number) {
    const newTree = tree.map((block, i) => {
      if (i !== blockIdx) return block;
      return { ...block, children: block.children.filter((_, ci) => ci !== childIdx) };
    });
    commit(newTree);
    setMenuPath(null);
  }

  function moveChildUp(blockIdx: number, childIdx: number) {
    if (childIdx === 0) return;
    const newTree = tree.map((block, i) => {
      if (i !== blockIdx) return block;
      const ch = [...block.children];
      [ch[childIdx - 1], ch[childIdx]] = [ch[childIdx], ch[childIdx - 1]];
      return { ...block, children: ch };
    });
    commit(newTree);
    setMenuPath(null);
  }

  function moveChildDown(blockIdx: number, childIdx: number) {
    const block = tree[blockIdx];
    if (childIdx === block.children.length - 1) return;
    const newTree = tree.map((b, i) => {
      if (i !== blockIdx) return b;
      const ch = [...b.children];
      [ch[childIdx], ch[childIdx + 1]] = [ch[childIdx + 1], ch[childIdx]];
      return { ...b, children: ch };
    });
    commit(newTree);
    setMenuPath(null);
  }

  function duplicateChild(blockIdx: number, childIdx: number) {
    const newTree = tree.map((block, i) => {
      if (i !== blockIdx) return block;
      const ch = [...block.children];
      ch.splice(childIdx + 1, 0, { ...ch[childIdx] });
      return { ...block, children: ch };
    });
    commit(newTree);
    setMenuPath(null);
  }

  // ─── Edit modal ───────────────────────────────────────────────────────────

  function openEdit(path: number[]) {
    const node = path.length === 1 ? tree[path[0]] : tree[path[0]].children[path[1]];
    setEditDraft({ ...node, children: node.children ?? [] });
    setEditPath(path);
    setMenuPath(null);
  }

  function saveEdit() {
    if (!editPath || !editDraft) return;
    let newTree: SegmentNode[];
    if (editPath.length === 1) {
      newTree = tree.map((n, i) => i === editPath[0] ? { ...editDraft, children: n.children } : n);
    } else {
      newTree = tree.map((block, i) => {
        if (i !== editPath[0]) return block;
        return {
          ...block,
          children: block.children.map((ch, ci) => ci === editPath[1] ? editDraft : ch),
        };
      });
    }
    commit(newTree);
    setEditPath(null);
    setEditDraft(null);
  }

  function cancelEdit() {
    setEditPath(null);
    setEditDraft(null);
  }

  function patchDraft(patch: Partial<SegmentNode>) {
    if (!editDraft) return;
    setEditDraft({ ...editDraft, ...patch });
  }

  // ─── Summary helpers ──────────────────────────────────────────────────────

  function unitLabel(unit: string) { return t(`unit_${unit}`, { defaultValue: unit }); }
  function intensityLabel(i: string) { return t(`intensity_${i}`, { defaultValue: i }); }

  function getSummary(seg: SegmentNode): string {
    switch (seg.segment_type) {
      case 'simple':
        return `${seg.value} ${unitLabel(seg.unit)} ${intensityLabel(seg.intensity).toLowerCase()}`;
      case 'interval': {
        const work = `${seg.work_value} ${unitLabel(seg.work_unit)} ${intensityLabel(seg.work_intensity).toLowerCase()}`;
        const rest = `${seg.rest_value} ${unitLabel(seg.rest_unit)} ${intensityLabel(seg.rest_intensity).toLowerCase()}`;
        return `${seg.repetitions} × ${work} / ${rest}`;
      }
      case 'rest':
        return seg.intensity
          ? `${seg.value} ${unitLabel(seg.unit)} — ${seg.intensity}`
          : `${seg.value} ${unitLabel(seg.unit)}`;
      case 'block':
        return t('segment_block_reps', { count: seg.repetitions }) +
          (seg.children.length > 0 ? ` · ${t('segment_block_children', { count: seg.children.length })}` : '');
      default:
        return '';
    }
  }

  // ─── Menu helpers ─────────────────────────────────────────────────────────

  function isMenuOpen(path: number[]) {
    return menuPath !== null &&
      menuPath.length === path.length &&
      menuPath.every((v, i) => v === path[i]);
  }

  function toggleMenu(path: number[]) {
    setMenuPath(isMenuOpen(path) ? null : path);
  }

  // ─── Segment row renderer ─────────────────────────────────────────────────

  function renderSegmentRow(
    seg: SegmentNode,
    path: number[],
    totalSiblings: number,
    isChild: boolean,
  ) {
    const idx = path[path.length - 1];
    return (
      <tr key={path.join('-')} className={`segment-row segment-row-${seg.segment_type}${isChild ? ' segment-row-child' : ''}`}>
        <td className="segment-order">{isChild ? `${path[0]+1}.${idx+1}` : idx + 1}</td>
        <td className="segment-reps">
          {seg.segment_type === 'interval' ? seg.repetitions : '—'}
        </td>
        <td className="segment-exercise">{getSummary(seg)}</td>
        <td className="segment-actions-cell">
          <div className="segment-menu-wrapper">
            <button type="button" className="btn-icon segment-menu-trigger" onClick={() => toggleMenu(path)}>⋮</button>
            {isMenuOpen(path) && (
              <div className="segment-dropdown">
                <button type="button" onClick={() => openEdit(path)}>{t('edit')}</button>
                <button type="button" onClick={() => isChild ? duplicateChild(path[0], idx) : duplicateRoot(idx)}>{t('segment_duplicate')}</button>
                {idx > 0 && (
                  <button type="button" onClick={() => isChild ? moveChildUp(path[0], idx) : moveRootUp(idx)}>{t('segment_move_up')}</button>
                )}
                {idx < totalSiblings - 1 && (
                  <button type="button" onClick={() => isChild ? moveChildDown(path[0], idx) : moveRootDown(idx)}>{t('segment_move_down')}</button>
                )}
                <button type="button" className="danger" onClick={() => isChild ? removeChild(path[0], idx) : removeRoot(idx)}>{t('delete')}</button>
              </div>
            )}
          </div>
        </td>
      </tr>
    );
  }

  // ─── Render ───────────────────────────────────────────────────────────────

  return (
    <div className="segment-builder">
      <h3>{t('segment_structure')}</h3>

      <div className="segment-actions">
        <button type="button" onClick={() => addRoot(makeSimple)}>+ {t('segment_add_simple')}</button>
        <button type="button" onClick={() => addRoot(makeInterval)}>+ {t('segment_add_interval')}</button>
        <button type="button" onClick={() => addRoot(makeRest)}>+ {t('segment_add_rest')}</button>
        <button type="button" onClick={() => addRoot(makeBlock)}>+ {t('segment_add_block')}</button>
      </div>

      {tree.length === 0 ? (
        <p className="segment-empty-msg">{t('segment_empty')}</p>
      ) : (
        <table className="segment-table">
          <thead>
            <tr>
              <th>#</th>
              <th>{t('segment_col_reps')}</th>
              <th>{t('segment_col_exercise')}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {tree.map((node, rootIdx) => (
              <>
                {node.segment_type === 'block' ? (
                  <>
                    <tr key={`block-${rootIdx}`} className="segment-row segment-row-block">
                      <td className="segment-order">{rootIdx + 1}</td>
                      <td className="segment-reps">{node.repetitions}×</td>
                      <td className="segment-exercise segment-block-header">
                        {t('segment_block')}
                        {node.children.length > 0 && (
                          <span className="segment-block-meta">
                            {t('segment_block_children', { count: node.children.length })}
                          </span>
                        )}
                      </td>
                      <td className="segment-actions-cell">
                        <div className="segment-menu-wrapper">
                          <button type="button" className="btn-icon segment-menu-trigger" onClick={() => toggleMenu([rootIdx])}>⋮</button>
                          {isMenuOpen([rootIdx]) && (
                            <div className="segment-dropdown">
                              <button type="button" onClick={() => openEdit([rootIdx])}>{t('edit')}</button>
                              <button type="button" onClick={() => duplicateRoot(rootIdx)}>{t('segment_duplicate')}</button>
                              {rootIdx > 0 && <button type="button" onClick={() => moveRootUp(rootIdx)}>{t('segment_move_up')}</button>}
                              {rootIdx < tree.length - 1 && <button type="button" onClick={() => moveRootDown(rootIdx)}>{t('segment_move_down')}</button>}
                              <button type="button" className="danger" onClick={() => removeRoot(rootIdx)}>{t('delete')}</button>
                            </div>
                          )}
                        </div>
                      </td>
                    </tr>
                    {node.children.map((child, childIdx) =>
                      renderSegmentRow(child, [rootIdx, childIdx], node.children.length, true)
                    )}
                    <tr key={`block-add-${rootIdx}`} className="segment-row-block-add">
                      <td colSpan={4}>
                        <div className="segment-block-add-row">
                          <button type="button" onClick={() => addChild(rootIdx, makeSimple)}>+ {t('segment_add_simple')}</button>
                          <button type="button" onClick={() => addChild(rootIdx, makeInterval)}>+ {t('segment_add_interval')}</button>
                          <button type="button" onClick={() => addChild(rootIdx, makeRest)}>+ {t('segment_add_rest')}</button>
                        </div>
                      </td>
                    </tr>
                  </>
                ) : (
                  renderSegmentRow(node, [rootIdx], tree.length, false)
                )}
              </>
            ))}
          </tbody>
        </table>
      )}

      {/* Edit modal */}
      {editPath !== null && editDraft && (
        <div className="modal-overlay" onClick={cancelEdit}>
          <div className="modal segment-edit-modal" onClick={(e) => e.stopPropagation()}>
            <h3>
              {editDraft.segment_type === 'simple' && t('segment_simple')}
              {editDraft.segment_type === 'interval' && t('segment_interval')}
              {editDraft.segment_type === 'rest' && t('segment_rest')}
              {editDraft.segment_type === 'block' && t('segment_block')}
            </h3>

            {/* Type switcher — not shown for block (you edit repetitions inline) */}
            {editDraft.segment_type !== 'block' && (
              <div className="form-group">
                <label>{t('segment_type_label')}</label>
                <div className="segment-type-toggle">
                  {(['simple', 'interval', 'rest'] as const).map((type) => (
                    <button
                      key={type}
                      type="button"
                      className={`btn btn-sm ${editDraft.segment_type === type ? 'btn-primary' : ''}`}
                      onClick={() => {
                        if (type === 'simple') patchDraft({ segment_type: 'simple', repetitions: 1, value: editDraft.work_value || 1, unit: editDraft.work_unit || 'km', intensity: editDraft.work_intensity || 'easy' });
                        else if (type === 'interval') patchDraft({ segment_type: 'interval', repetitions: editDraft.repetitions || 3, work_value: editDraft.value || 1, work_unit: editDraft.unit || 'min', work_intensity: editDraft.intensity || 'fast' });
                        else patchDraft({ segment_type: 'rest', repetitions: 1, value: editDraft.value || 1, unit: 'min', intensity: '' });
                      }}
                    >
                      {t(`segment_${type}`)}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {editDraft.segment_type === 'block' && (
              <div className="form-group">
                <label>{t('segment_repetitions')}</label>
                <input type="number" min={2} value={editDraft.repetitions}
                  onChange={(e) => patchDraft({ repetitions: Number(e.target.value) })} />
              </div>
            )}

            {editDraft.segment_type === 'simple' && (
              <>
                <div className="form-group">
                  <label>{t('segment_value')}</label>
                  <div className="segment-inline-fields">
                    <input type="number" min={0} step="any" value={editDraft.value}
                      onChange={(e) => patchDraft({ value: Number(e.target.value) })} />
                    <select value={editDraft.unit} onChange={(e) => patchDraft({ unit: e.target.value })}>
                      {UNITS.map(u => <option key={u} value={u}>{t(`unit_${u}`)}</option>)}
                    </select>
                  </div>
                </div>
                <div className="form-group">
                  <label>{t('segment_intensity_label')}</label>
                  <select value={editDraft.intensity} onChange={(e) => patchDraft({ intensity: e.target.value })}>
                    {INTENSITIES.map(i => <option key={i} value={i}>{t(`intensity_${i}`)}</option>)}
                  </select>
                </div>
              </>
            )}

            {editDraft.segment_type === 'interval' && (
              <>
                <div className="form-group">
                  <label>{t('segment_repetitions')}</label>
                  <input type="number" min={1} value={editDraft.repetitions}
                    onChange={(e) => patchDraft({ repetitions: Number(e.target.value) })} />
                </div>
                <div className="form-group">
                  <label>{t('segment_work')}</label>
                  <div className="segment-inline-fields">
                    <input type="number" min={0} step="any" value={editDraft.work_value}
                      onChange={(e) => patchDraft({ work_value: Number(e.target.value) })} />
                    <select value={editDraft.work_unit} onChange={(e) => patchDraft({ work_unit: e.target.value })}>
                      {UNITS.map(u => <option key={u} value={u}>{t(`unit_${u}`)}</option>)}
                    </select>
                    <select value={editDraft.work_intensity} onChange={(e) => patchDraft({ work_intensity: e.target.value })}>
                      {INTENSITIES.map(i => <option key={i} value={i}>{t(`intensity_${i}`)}</option>)}
                    </select>
                  </div>
                </div>
                <div className="form-group">
                  <label>{t('segment_rest')}</label>
                  <div className="segment-inline-fields">
                    <input type="number" min={0} step="any" value={editDraft.rest_value}
                      onChange={(e) => patchDraft({ rest_value: Number(e.target.value) })} />
                    <select value={editDraft.rest_unit} onChange={(e) => patchDraft({ rest_unit: e.target.value })}>
                      {UNITS.map(u => <option key={u} value={u}>{t(`unit_${u}`)}</option>)}
                    </select>
                    <select value={editDraft.rest_intensity} onChange={(e) => patchDraft({ rest_intensity: e.target.value })}>
                      {INTENSITIES.map(i => <option key={i} value={i}>{t(`intensity_${i}`)}</option>)}
                    </select>
                  </div>
                </div>
              </>
            )}

            {editDraft.segment_type === 'rest' && (
              <>
                <div className="form-group">
                  <label>{t('segment_value')}</label>
                  <div className="segment-inline-fields">
                    <input type="number" min={0} step="any" value={editDraft.value}
                      onChange={(e) => patchDraft({ value: Number(e.target.value) })} />
                    <select value={editDraft.unit} onChange={(e) => patchDraft({ unit: e.target.value })}>
                      {(['min', 'sec'] as const).map(u => <option key={u} value={u}>{t(`unit_${u}`)}</option>)}
                    </select>
                  </div>
                </div>
                <div className="form-group">
                  <label>{t('field_notes')} ({t('optional', { defaultValue: 'opcional' })})</label>
                  <input type="text" value={editDraft.intensity}
                    placeholder="ej: trote suave, caminata..."
                    onChange={(e) => patchDraft({ intensity: e.target.value })} />
                </div>
              </>
            )}

            <div className="modal-actions">
              <button type="button" className="btn btn-sm" onClick={cancelEdit}>{t('cancel')}</button>
              <button type="button" className="btn btn-sm btn-primary" onClick={saveEdit}>{t('save')}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Verify the frontend builds**

```bash
cd ~/Desktop/FitReg/FitRegFE
npm run build 2>&1 | grep -E "error|Error" | head -20
```

Expected: no build errors. If TypeScript complains about `children: undefined as never`, change that to an explicit `delete` or cast: `{ ...node, children: undefined } as WorkoutSegment`.

- [ ] **Step 3: Run the dev server and manually verify**

```bash
npm run dev
```

Open http://localhost:5173, log in, go to create a workout. Verify:
- Four add buttons: Simple, Intervalo, Descanso, Bloque
- Adding a block creates a row with its own add buttons below
- Adding a segment inside a block renders it indented
- Editing a block opens modal with only repetitions field
- Editing a rest segment shows duration + optional note
- Saving a workout with blocks sends the correct flat array to the API (check Network tab — should have `temp_id: -1` on block, `parent_id: -1` on children)

- [ ] **Step 4: Commit**

```bash
git add src/components/SegmentBuilder.tsx
git commit -m "feat: SegmentBuilder supports block and rest segment types"
```

---

## Task 9: SegmentDisplay — Block and Rest Rendering

**Files:**
- Modify: `FitRegFE/src/components/SegmentDisplay.tsx`

- [ ] **Step 1: Rewrite SegmentDisplay**

Replace the entire file:

```tsx
import { useTranslation } from 'react-i18next';
import type { WorkoutSegment, SegmentNode } from '../types';

interface SegmentDisplayProps {
  segments: WorkoutSegment[];
}

export default function SegmentDisplay({ segments }: SegmentDisplayProps) {
  const { t } = useTranslation();

  function unitLabel(unit: string) { return t(`unit_${unit}`, { defaultValue: unit }); }
  function intensityLabel(i: string) { return t(`intensity_${i}`, { defaultValue: i }); }
  function intensityClass(i: string) { return `intensity-${i}`; }

  // Build tree from flat array (same logic as SegmentBuilder)
  const roots = segments
    .filter(s => !s.parent_id)
    .sort((a, b) => a.order_index - b.order_index);

  function getChildren(parentId: number): WorkoutSegment[] {
    return segments
      .filter(s => s.parent_id === parentId)
      .sort((a, b) => a.order_index - b.order_index);
  }

  function renderSegment(seg: WorkoutSegment, index: number, prefix = '') {
    const label = `${prefix}${index + 1}.`;
    switch (seg.segment_type) {
      case 'simple':
        return (
          <div key={seg.id ?? index} className="segment-display-item">
            <span className="segment-display-number">{label}</span>
            <span>
              {seg.value} {unitLabel(seg.unit)}{' '}
              <span className={intensityClass(seg.intensity)}>
                {intensityLabel(seg.intensity).toLowerCase()}
              </span>
            </span>
          </div>
        );
      case 'interval':
        return (
          <div key={seg.id ?? index} className="segment-display-item">
            <span className="segment-display-number">{label}</span>
            <span>
              {seg.repetitions} &times;{' '}
              {seg.work_value} {unitLabel(seg.work_unit)}{' '}
              <span className={intensityClass(seg.work_intensity)}>
                {intensityLabel(seg.work_intensity).toLowerCase()}
              </span>
              <span className="segment-display-separator"> / </span>
              {seg.rest_value} {unitLabel(seg.rest_unit)}{' '}
              <span className={intensityClass(seg.rest_intensity)}>
                {intensityLabel(seg.rest_intensity).toLowerCase()}
              </span>
            </span>
          </div>
        );
      case 'rest':
        return (
          <div key={seg.id ?? index} className="segment-display-item segment-display-rest">
            <span className="segment-display-number">{label}</span>
            <span className="intensity-easy">
              {t('segment_rest')}: {seg.value} {unitLabel(seg.unit)}
              {seg.intensity ? ` — ${seg.intensity}` : ''}
            </span>
          </div>
        );
      case 'block': {
        const children = seg.id ? getChildren(seg.id) : [];
        return (
          <div key={seg.id ?? index} className="segment-display-block">
            <div className="segment-display-block-header">
              <span className="segment-display-number">{label}</span>
              <span>
                {t('segment_block')} &times; {seg.repetitions}
              </span>
            </div>
            <div className="segment-display-block-body">
              {children.map((child, ci) => renderSegment(child, ci, `${index + 1}.`))}
            </div>
          </div>
        );
      }
      default:
        return null;
    }
  }

  return (
    <div className="segment-display">
      {roots.map((seg, i) => renderSegment(seg, i))}
    </div>
  );
}
```

- [ ] **Step 2: Add CSS for block display**

In `FitRegFE/src/App.css`, find the `.segment-display` section and add:

```css
.segment-display-block {
  border: 1.5px solid #8b5cf6;
  border-radius: 8px;
  margin-bottom: 8px;
  overflow: hidden;
}

.segment-display-block-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  background: rgba(139, 92, 246, 0.08);
  font-weight: 600;
  font-size: 14px;
  color: #7c3aed;
  border-bottom: 1px solid rgba(139, 92, 246, 0.2);
}

.segment-display-block-body {
  padding: 4px 8px 4px 20px;
}

.segment-display-block-body .segment-display-item {
  padding-left: 4px;
  border-left: 2px solid #c4b5fd;
  margin-left: 4px;
}

.segment-display-rest {
  opacity: 0.8;
}
```

Also add CSS for the block builder rows in `App.css`, find `.segment-table` and add:

```css
.segment-row-block {
  background: rgba(139, 92, 246, 0.05);
  font-weight: 600;
}

.segment-row-child td {
  padding-left: 24px;
  font-size: 13px;
  color: var(--text-secondary, #666);
  border-left: 2px solid #c4b5fd;
}

.segment-row-block-add td {
  padding: 4px 8px 8px 24px;
  border-bottom: 2px solid #8b5cf6;
}

.segment-block-add-row {
  display: flex;
  gap: 8px;
}

.segment-block-add-row button {
  font-size: 12px;
  padding: 3px 10px;
  border: 1.5px dashed #8b5cf6;
  background: transparent;
  color: #7c3aed;
  border-radius: 4px;
  cursor: pointer;
}

.segment-block-meta {
  font-size: 12px;
  font-weight: normal;
  color: #9f7aea;
  margin-left: 6px;
}
```

- [ ] **Step 3: Build and verify**

```bash
npm run build 2>&1 | grep -E "error|Error" | head -20
```

Run dev server and navigate to a workout with segments to verify SegmentDisplay renders correctly for existing (simple/interval) workouts. Then create and save a workout with a block and verify it displays with the purple border and indented children.

- [ ] **Step 4: Commit**

```bash
git add src/components/SegmentDisplay.tsx src/App.css
git commit -m "feat: SegmentDisplay renders block and rest segment types"
```

---

## Task 10: End-to-End Smoke Test

- [ ] **Step 1: Start the backend**

```bash
cd ~/Desktop/FitReg/FitRegAPI
export $(cat .env | xargs)
go run main.go
```

- [ ] **Step 2: Start the frontend**

```bash
cd ~/Desktop/FitReg/FitRegFE
npm run dev
```

- [ ] **Step 3: Create the fartlek workout from the spec**

Log in, go to create workout, set type = fartlek, then build:
1. `+ Simple` → 15 min suave
2. `+ Bloque` → set repetitions = 3
   - Inside the block: `+ Simple` → 1 min rápido
   - `+ Descanso` → 1 min
   - `+ Simple` → 2 min rápido
   - `+ Descanso` → 1 min
   - `+ Simple` → 1 min rápido
   - `+ Descanso` → 3 min (nota: "entre bloques")
3. `+ Simple` → 15 min suave

Save the workout. Expected: no error, workout is saved.

- [ ] **Step 4: Verify in DB**

```bash
mysql -u root fitreg -e "
SELECT id, parent_id, order_index, segment_type, value, unit, intensity
FROM workout_segments
WHERE workout_id = (SELECT MAX(id) FROM workouts)
ORDER BY COALESCE(parent_id, id), order_index;"
```

Expected: 1 root simple, 1 root block, 6 child segments under the block (matching parent_id = block id), 1 root simple.

- [ ] **Step 5: Reload and verify display**

Navigate to the workout detail. SegmentDisplay should show: calentamiento, bloque×3 (with 6 children indented, purple border), vuelta calma.

- [ ] **Step 6: Update mvp-review.md**

In `FitRegAPI/docs/mvp-review.md`, in the Deuda técnica section, mark the segment blocks item as resolved:

This is a new feature, so add under Implementado en sesiones anteriores:
```
- [x] **Bloques de segmentos estructurados** — `rest` y `block` como tipos de segmento. Bloques se repiten N veces. UI inline en SegmentBuilder.
```

- [ ] **Step 7: Commit both repos**

```bash
cd ~/Desktop/FitReg/FitRegAPI
git add docs/mvp-review.md
git commit -m "docs: mark structured workout blocks as implemented"

cd ~/Desktop/FitReg/FitRegFE
git add -A
git commit -m "feat: structured workout blocks end-to-end complete"
```
