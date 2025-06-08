-- name: CreateJobPosting :exec
INSERT INTO job_postings (
    id,
    title,
    company_name,
    prefecture_code,
    prefecture_name,
    municipality,
    summary_url,
    job_type,
    salary_min_amount,
    salary_max_amount,
    salary_unit,
    salary_currency,
    salary_is_fixed,
    posted_at,
    job_name,
    holiday_policy,
    raise,
    bonus,
    description,
    requirements,
    holidays_per_year,
    work_hours,
    benefits
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8,
    $9, $10, $11, $12, $13, $14,
    $15, $16, $17, $18, $19, $20, $21, $22, $23
);

-- name: GetJobPostingByID :one
SELECT
    id,
    title,
    company_name,
    prefecture_code,
    prefecture_name,
    municipality,
    summary_url,
    job_type,
    salary_min_amount,
    salary_max_amount,
    salary_unit,
    salary_currency,
    salary_is_fixed,
    posted_at,
    job_name,
    holiday_policy,
    raise,
    bonus,
    description,
    requirements,
    holidays_per_year,
    work_hours,
    benefits
FROM job_postings
WHERE id = $1;
