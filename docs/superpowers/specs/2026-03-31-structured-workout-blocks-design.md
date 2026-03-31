# Structured Workout Blocks — Design Spec

_Date: 2026-03-31_

## Context

Coaches need to express structured running workouts like fartlek where a pattern of segments repeats N times (e.g. "3 blocks of 1min fast / 1min rest / 2min fast / 1min rest / 1min fast / 3min recovery"). Today this requires manually duplicating identical rows in the segment list — 17 rows for a workout that is conceptually 4 items.

The most complex workout we need to support is: `15min warmup + 3 × [1min fast, 1min rest, 2min fast, 1min rest, 1min fast, 3min recovery] + 15min cooldown`. One level of nesting is sufficient.

---

## Decisions

| Aspect | Decision |
|--------|----------|
| Structure | Grouped blocks with repetition count, 1 level deep |
| New segment types | `rest` (duration + optional note) and `block` (container with N repetitions) |
| Block UX | Inline — block appears in the list and expands; segments are added directly inside it |
| Max depth | 1 level: a block contains segments, not other blocks |
| API serialization | Flat array with `parent_id`; frontend builds the tree |
| Backwards compatibility | Additive only — existing segments unaffected |

---

## Data Model

### New segment types

Extend the `segment_type` ENUM with two new values:

- **`rest`** — a recovery period between efforts or blocks
- **`block`** — a container that groups child segments and repeats them N times

### DB migration

Add `parent_id` column to all segment tables:

```sql
ALTER TABLE workout_segments
  ADD COLUMN parent_id BIGINT NULL REFERENCES workout_segments(id) ON DELETE CASCADE,
  MODIFY COLUMN segment_type ENUM('simple','interval','rest','block') NOT NULL;
```

Same change applies to:
- `workout_template_segments`
- `weekly_template_day_segments`

(Note: `assigned_workout_segments` was removed in the unified workouts migration — there is only `workout_segments` now.)

### Field usage per type

| Field | `simple` | `interval` | `rest` | `block` |
|-------|----------|------------|--------|---------|
| `segment_type` | `simple` | `interval` | `rest` | `block` |
| `repetitions` | — | N reps | — | N times block repeats |
| `value` / `unit` | distance or duration | — | duration (required) | — |
| `intensity` | easy/moderate/fast/sprint | — | optional free-text note | — |
| `work_*` | — | work phase | — | — |
| `rest_*` | — | rest phase | — | — |
| `parent_id` | NULL or block ID | NULL or block ID | NULL or block ID | always NULL |
| `order_index` | position in parent scope | position in parent scope | position in parent scope | position at root level |

**Rules:**
- `block` segments always have `parent_id = NULL` (they are always at root level)
- Child segments have `parent_id` pointing to their containing block's `id`
- `order_index` is scoped to the parent context: root segments are ordered among themselves; children within a block are ordered among themselves

### Go model change

```go
type WorkoutSegment struct {
    ID            int64   `json:"id"`
    ParentID      *int64  `json:"parent_id"`       // new field; nil = root level
    OrderIndex    int     `json:"order_index"`
    SegmentType   string  `json:"segment_type"`    // "simple"|"interval"|"rest"|"block"
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

The API always serializes segments as a flat array. Tree construction is the frontend's responsibility.

---

## API

### Read (GET)

The API returns segments as a **flat array** with `parent_id`. The frontend builds the tree. The backend does not nest segments.

Example response for the workout described in context:

```json
[
  { "id": 1, "parent_id": null, "order_index": 0, "segment_type": "simple", "value": 15, "unit": "min", "intensity": "easy" },
  { "id": 2, "parent_id": null, "order_index": 1, "segment_type": "block", "repetitions": 3 },
  { "id": 3, "parent_id": 2,    "order_index": 0, "segment_type": "simple", "value": 1, "unit": "min", "intensity": "fast" },
  { "id": 4, "parent_id": 2,    "order_index": 1, "segment_type": "rest",   "value": 1, "unit": "min" },
  { "id": 5, "parent_id": 2,    "order_index": 2, "segment_type": "simple", "value": 2, "unit": "min", "intensity": "fast" },
  { "id": 6, "parent_id": 2,    "order_index": 3, "segment_type": "rest",   "value": 1, "unit": "min" },
  { "id": 7, "parent_id": 2,    "order_index": 4, "segment_type": "simple", "value": 1, "unit": "min", "intensity": "fast" },
  { "id": 8, "parent_id": 2,    "order_index": 5, "segment_type": "rest",   "value": 3, "unit": "min", "intensity": "entre bloques" },
  { "id": 9, "parent_id": null, "order_index": 2, "segment_type": "simple", "value": 15, "unit": "min", "intensity": "easy" }
]
```

### Write (POST/PUT)

The client sends the same flat array. The repository deletes all existing segments for that workout and re-inserts (current behavior — unchanged). Segments do not carry a persistent `id` in write requests; `parent_id` in the write payload uses a **client-side temporary ID** (negative integers, e.g. `-1`, `-2`) for blocks. The repository resolves this in two passes: first insert block segments and capture their real DB IDs, then insert children substituting the temporary `parent_id` with the real one.

**Validation:**
- A `block` segment must not have a `parent_id`
- A segment with `parent_id` must reference a `block` segment in the same workout
- `rest` requires `value > 0` and `unit` in `('min', 'sec')`
- `block` requires `repetitions >= 2`

---

## Repository

### `ReplaceSegments(workoutID int64, segments []WorkoutSegment)`

Current behavior: delete all + bulk insert. With blocks, insertion uses two passes:

1. **Pass 1 — root segments:** Insert all segments where `parent_id` is nil (including blocks). Capture the real DB ID assigned to each block, keyed by the block's client-side temporary ID (negative int sent by the frontend).
2. **Pass 2 — children:** Insert all segments where `parent_id` is a negative temporary ID. Substitute the temporary `parent_id` with the real DB ID captured in Pass 1.

No other changes to the repository interface.

---

## Frontend

### TypeScript type change

```typescript
interface WorkoutSegment {
  id?: number;
  parent_id?: number | null;            // new
  order_index: number;
  segment_type: 'simple' | 'interval' | 'rest' | 'block';  // rest and block are new
  repetitions: number;
  value: number;
  unit: 'km' | 'm' | 'min' | 'sec';
  intensity: string;
  work_value: number;
  work_unit: string;
  work_intensity: string;
  rest_value: number;
  rest_unit: string;
  rest_intensity: string;
}
```

### SegmentBuilder — key changes

**Internal state:** The component works with a tree structure internally:

```typescript
type SegmentNode = WorkoutSegment & { children?: SegmentNode[] }
```

On load: flat array → tree (group children under their block parent).
On save: tree → flat array (passed to `onChange`).

**Rendering:**
- Root-level nodes render as table rows (current behavior for simple/interval/rest)
- Root-level `block` nodes render as a bordered container with:
  - Header: `⟳ Bloque × N` with editable repetitions field
  - Body: indented list of child segments
  - Footer: `+ Simple  + Intervalo  + Descanso` buttons scoped to the block

**Root-level add buttons:** `+ Simple  + Intervalo  + Descanso  + Bloque`

**Reordering:**
- Move up/down operates within the same scope (root ↔ root, child ↔ child within same block)
- A child cannot be moved to root level via reorder (must delete and re-add)

**`getSummary()` additions:**
- `rest`: `"3 min recuperación"` or `"3 min"` if no note
- `block`: `"Bloque ×3 (5 segmentos)"` — summary shows repetitions and child count

### SegmentDisplay — key changes

Add rendering for `block` nodes: show the block header (`⟳ Bloque × N`) and indent child segments beneath it using the same block-group visual (purple border, left indent bar).

---

## Backwards Compatibility

- `parent_id = NULL` and `segment_type IN ('simple', 'interval')` continues to work exactly as before
- Existing workouts require no migration — they have no `parent_id` and use existing types
- The DB migration is additive: `ADD COLUMN parent_id` (nullable) + `MODIFY COLUMN segment_type` (ENUM extension)
- The frontend handles the old flat shape transparently — all existing segments have `parent_id = null` and are treated as root-level nodes

---

## Out of scope

- Sub-blocks (blocks inside blocks) — 1 level is sufficient for the target use case
- Reordering a child segment to root level via drag — too complex, low need
- Displaying segment totals per block (e.g. "total: 9 min work per block") — future enhancement
