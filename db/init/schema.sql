-- Enum types (unchanged)
CREATE TYPE salary_type AS ENUM (
  'Hourly',
  'Daily',
  'Monthly',
  'Yearly'
);

CREATE TYPE job_type AS ENUM (
  'Unknown',
  'FullTime',
  'PartTime',
  'Contract',
  'Temporary',
  'Freelance',
  'Internship',
  'Other'
);

CREATE TYPE holiday_policy AS ENUM (
  'CompleteTwoDaysAWeek',
  'TwoDaysAWeek',
  'OneDayAWeek',
  'ShiftSystem',
  'UnknownHoliday'
);

CREATE TYPE workplace_type AS ENUM (
  'Onsite',
  'Remote',
  'Hybrid',
  'FullRemote',
  'UnknownWorkplace'
);

-- Companies table
CREATE TABLE companies (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  headquarters_prefecture_code TEXT NOT NULL,
  headquarters_prefecture_name TEXT NOT NULL,
  headquarters_municipality TEXT NOT NULL,
  headquarters_raw TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  -- Unique constraint to prevent duplicate companies
  CONSTRAINT unique_company_headquarters UNIQUE (name, headquarters_raw)
);

-- Locations table
CREATE TABLE locations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  prefecture_code TEXT NOT NULL,
  prefecture_name TEXT NOT NULL,
  municipality TEXT NOT NULL,
  raw_location TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  -- Unique constraint for locations
  CONSTRAINT unique_location UNIQUE (prefecture_code, municipality, raw_location)
);

-- Job postings table (normalized)
CREATE TABLE job_postings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  location_id UUID NOT NULL REFERENCES locations(id) ON DELETE RESTRICT,

  -- Basic job info
  title TEXT NOT NULL,
  job_name TEXT NOT NULL,
  summary_url TEXT NOT NULL,
  job_type job_type NOT NULL DEFAULT 'Unknown',

  -- Salary information
  salary_min_amount BIGINT NOT NULL,
  salary_max_amount BIGINT NOT NULL,
  salary_unit salary_type NOT NULL,
  salary_is_fixed BOOLEAN NOT NULL DEFAULT false,
  raise INTEGER,
  bonus INTEGER,

  -- Work details
  description TEXT NOT NULL,
  requirements TEXT NOT NULL,
  workplace_type workplace_type NOT NULL DEFAULT 'UnknownWorkplace',
  work_hours TEXT NOT NULL,

  -- Holiday information
  holidays_per_year INTEGER,
  holiday_policy holiday_policy NOT NULL DEFAULT 'UnknownHoliday',

  -- Timestamps
  posted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  -- Constraints
  CONSTRAINT check_salary_range CHECK (salary_min_amount <= salary_max_amount),
  CONSTRAINT check_holidays_range CHECK (holidays_per_year IS NULL OR (holidays_per_year >= 0 AND holidays_per_year <= 366))
);

-- Benefits table (separate for better organization)
CREATE TABLE job_benefits (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  job_posting_id UUID NOT NULL REFERENCES job_postings(id) ON DELETE CASCADE,

  -- Insurance and allowances
  social_insurance BOOLEAN NOT NULL DEFAULT false,
  transport_allowance BOOLEAN NOT NULL DEFAULT false,
  housing_allowance BOOLEAN NOT NULL DEFAULT false,
  company_housing BOOLEAN NOT NULL DEFAULT false,
  rent_subsidy BOOLEAN NOT NULL DEFAULT false,
  meal_allowance BOOLEAN NOT NULL DEFAULT false,
  cafeteria_provided BOOLEAN NOT NULL DEFAULT false,

  -- Career development
  training_support BOOLEAN NOT NULL DEFAULT false,
  certification_support BOOLEAN NOT NULL DEFAULT false,

  -- Work-life balance
  paid_leave BOOLEAN NOT NULL DEFAULT false,
  special_leave BOOLEAN NOT NULL DEFAULT false,
  flex_time BOOLEAN NOT NULL DEFAULT false,
  short_working_hours BOOLEAN NOT NULL DEFAULT false,

  -- Family support
  childcare_support BOOLEAN NOT NULL DEFAULT false,
  maternity_leave BOOLEAN NOT NULL DEFAULT false,
  parental_leave BOOLEAN NOT NULL DEFAULT false,
  elder_care_support BOOLEAN NOT NULL DEFAULT false,

  -- Retirement
  retirement_plan BOOLEAN NOT NULL DEFAULT false,

  -- Raw benefits text for additional info
  raw_benefits TEXT NOT NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  -- One-to-one relationship with job_postings
  CONSTRAINT unique_job_benefits UNIQUE (job_posting_id)
);
