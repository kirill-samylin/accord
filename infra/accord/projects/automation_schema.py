"""Schema6 is an authority boundary: incomplete declared storage is never repaired."""
TABLES = ('automation_policies', 'automation_policy_revisions', 'workflows',
          'workflow_steps', 'workflow_plans', 'workflow_questions',
          'workflow_attestations', 'execution_capabilities', 'workflow_reservations')
INDEXES = {
    'automation_step_execution': "CREATE UNIQUE INDEX automation_step_execution ON workflow_steps(json_extract(data,'$.workflow_id'),json_extract(data,'$.sequence'),json_extract(data,'$.attempt_no'))",
    'automation_cap_attempt': "CREATE UNIQUE INDEX automation_cap_attempt ON execution_capabilities(json_extract(data,'$.run_id'),json_extract(data,'$.generation'))",
}


def normalized(sql):
    # SQLite JSON paths are case and whitespace sensitive inside SQL literals.
    # The migration emits this canonical DDL; only outer whitespace is harmless.
    return sql.strip()


def validate_automation(connection):
    for table in TABLES:
        connection.execute('SELECT key,data FROM ' + table + ' LIMIT 0')
    if connection.execute('SELECT id,version FROM automation_metadata').fetchall() != [(1, 1)]:
        raise RuntimeError('automation metadata incomplete')
    for name, expected in INDEXES.items():
        rows = connection.execute("SELECT sql FROM sqlite_master WHERE type='index' AND name=?", (name,)).fetchall()
        if len(rows) != 1 or not isinstance(rows[0][0], str) or normalized(rows[0][0]) != normalized(expected):
            raise RuntimeError('automation execution constraint incomplete or changed')
