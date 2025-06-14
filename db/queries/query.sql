-- Company queries
-- name: CreateCompany :one
INSERT INTO companies (
    name,
    headquarters_prefecture_code,
    headquarters_prefecture_name,
    headquarters_municipality,
    headquarters_raw
) VALUES (
    $1, $2, $3, $4, $5
) ON CONFLICT (name, headquarters_raw)
DO UPDATE SET updated_at = now()
RETURNING id;

-- name: GetCompanyByName :one
SELECT id, name, headquarters_prefecture_code, headquarters_prefecture_name, headquarters_municipality, headquarters_raw, created_at FROM companies WHERE name = $1 LIMIT 1;

-- Location queries
-- name: CreateLocation :one
INSERT INTO locations (
    prefecture_code,
    prefecture_name,
    municipality,
    raw_location
) VALUES (
    $1, $2, $3, $4
) ON CONFLICT (prefecture_code, municipality, raw_location)
DO UPDATE SET created_at = locations.created_at
RETURNING id;

-- name: GetLocationByPrefectureAndMunicipality :one
SELECT id, prefecture_code, prefecture_name, municipality, raw_location, created_at FROM locations WHERE prefecture_code = $1 AND municipality = $2 LIMIT 1;

-- Job posting queries
-- name: CreateJobPosting :one
INSERT INTO job_postings (
    company_id,
    location_id,
    title,
    job_name,
    summary_url,
    job_type,
    salary_min_amount,
    salary_max_amount,
    salary_unit,
    salary_is_fixed,
    raise,
    bonus,
    description,
    requirements,
    workplace_type,
    work_hours,
    holidays_per_year,
    holiday_policy,
    posted_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15, $16, $17, $18, $19
) RETURNING id;

-- name: GetJobPostingByID :one
SELECT
    jp.id,
    c.name AS company_name,
    l.prefecture_code,
    l.prefecture_name,
    l.municipality,
    l.raw_location,
    jp.title,
    jp.job_name,
    jp.summary_url,
    jp.job_type,
    jp.salary_min_amount,
    jp.salary_max_amount,
    jp.salary_unit,
    jp.salary_is_fixed,
    jp.raise,
    jp.bonus,
    jp.description,
    jp.requirements,
    jp.workplace_type,
    jp.work_hours,
    jp.holidays_per_year,
    jp.holiday_policy,
    jp.posted_at,
    jp.created_at
FROM job_postings jp
JOIN companies c ON jp.company_id = c.id
JOIN locations l ON jp.location_id = l.id
WHERE jp.id = $1 LIMIT 1;

-- Benefits queries
-- name: CreateJobBenefits :exec
INSERT INTO job_benefits (
    job_posting_id,
    social_insurance,
    transport_allowance,
    housing_allowance,
    company_housing,
    rent_subsidy,
    meal_allowance,
    cafeteria_provided,
    training_support,
    certification_support,
    paid_leave,
    special_leave,
    flex_time,
    short_working_hours,
    childcare_support,
    maternity_leave,
    parental_leave,
    elder_care_support,
    retirement_plan,
    raw_benefits
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
);

-- name: GetJobBenefitByJobPostingID :one
SELECT
    id,
    job_posting_id,
    social_insurance,
    transport_allowance,
    housing_allowance,
    company_housing,
    rent_subsidy,
    meal_allowance,
    cafeteria_provided,
    training_support,
    certification_support,
    paid_leave,
    special_leave,
    flex_time,
    short_working_hours,
    childcare_support,
    maternity_leave,
    parental_leave,
    elder_care_support,
    retirement_plan,
    raw_benefits,
    created_at
FROM job_benefits
WHERE job_posting_id = $1 LIMIT 1;
