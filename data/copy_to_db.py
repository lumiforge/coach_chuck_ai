import json
import os
from pathlib import Path

import psycopg

INPUT_FILE = Path("exercises.json")

DB_CONFIG = {
    "host": os.getenv("PGHOST", "localhost"),
    "port": int(os.getenv("PGPORT", "5432")),
    "user": os.getenv("PGUSER", "postgres"),
    "password": os.getenv("PGPASSWORD", "postgres"),
    "dbname": os.getenv("PGDATABASE", "coach_chuck_db"),
}


def load_json(path: Path) -> dict:
    with path.open("r", encoding="utf-8") as f:
        return json.load(f)


def normalize_text(value: str) -> str:
    return " ".join(value.strip().lower().split())


def normalize_difficulty(value: str) -> str:
    difficulty = normalize_text(value)
    allowed = {"beginner", "intermediate", "advanced"}
    if difficulty not in allowed:
        raise ValueError(f"Некорректный difficulty: {value!r}")
    return difficulty


def split_multi_value_field(value: str) -> list[str]:
    if not value:
        return []

    parts = []
    for item in value.split(","):
        normalized = normalize_text(item)
        if normalized:
            parts.append(normalized)

    seen = set()
    result = []
    for item in parts:
        if item not in seen:
            seen.add(item)
            result.append(item)

    return result


def upsert_body_part(cur, name: str) -> int:
    cur.execute(
        """
        INSERT INTO body_parts (name)
        VALUES (%s)
        ON CONFLICT (name) DO UPDATE
        SET name = EXCLUDED.name
        RETURNING id
        """,
        (name,),
    )
    return cur.fetchone()[0]


def upsert_equipment(cur, name: str) -> int:
    cur.execute(
        """
        INSERT INTO equipment (name)
        VALUES (%s)
        ON CONFLICT (name) DO UPDATE
        SET name = EXCLUDED.name
        RETURNING id
        """,
        (name,),
    )
    return cur.fetchone()[0]


def upsert_exercise(cur, name: str, difficulty: str, description: str) -> int:
    cur.execute(
        """
        INSERT INTO exercises (name, difficulty, description)
        VALUES (%s, %s, %s)
        ON CONFLICT (name) DO UPDATE
        SET difficulty = EXCLUDED.difficulty,
            description = EXCLUDED.description
        RETURNING id
        """,
        (name.strip(), difficulty, description.strip()),
    )
    return cur.fetchone()[0]


def link_exercise_body_part(cur, exercise_id: int, body_part_id: int) -> None:
    cur.execute(
        """
        INSERT INTO exercise_body_parts (exercise_id, body_part_id)
        VALUES (%s, %s)
        ON CONFLICT DO NOTHING
        """,
        (exercise_id, body_part_id),
    )


def link_exercise_equipment(cur, exercise_id: int, equipment_id: int) -> None:
    cur.execute(
        """
        INSERT INTO exercise_equipment (exercise_id, equipment_id)
        VALUES (%s, %s)
        ON CONFLICT DO NOTHING
        """,
        (exercise_id, equipment_id),
    )


def main():
    data = load_json(INPUT_FILE)
    items = data["data"]

    with psycopg.connect(**DB_CONFIG) as conn:
        with conn.cursor() as cur:
            for item in items:
                name = item["name"].strip()
                difficulty = normalize_difficulty(item["difficulty"])
                description = item["description"].strip()

                body_parts = split_multi_value_field(item.get("body_part", ""))
                equipment_items = split_multi_value_field(item.get("equipment", ""))

                exercise_id = upsert_exercise(
                    cur=cur,
                    name=name,
                    difficulty=difficulty,
                    description=description,
                )

                for body_part_name in body_parts:
                    body_part_id = upsert_body_part(cur, body_part_name)
                    link_exercise_body_part(cur, exercise_id, body_part_id)

                for equipment_name in equipment_items:
                    equipment_id = upsert_equipment(cur, equipment_name)
                    link_exercise_equipment(cur, exercise_id, equipment_id)

        conn.commit()

    print("Импорт завершён")


if __name__ == "__main__":
    main()