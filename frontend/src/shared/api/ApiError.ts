export interface ApiFieldError {
  field: string;
  message: string;
}

/** Error built from an RFC 9457 Problem Details response (docs/06-api.md). */
export class ApiError extends Error {
  readonly status: number;
  readonly title: string;
  readonly detail?: string;
  readonly fields: ApiFieldError[];

  constructor(status: number, title: string, detail?: string, fields: ApiFieldError[] = []) {
    super(detail ? `${title}: ${detail}` : title);
    this.name = 'ApiError';
    this.status = status;
    this.title = title;
    this.detail = detail;
    this.fields = fields;
  }

  get isUnauthorized(): boolean {
    return this.status === 401;
  }
}

/** The response was received but did not match the expected schema. */
export class ApiContractError extends Error {
  constructor(path: string, cause: unknown) {
    super(`Unexpected response shape from ${path}`, { cause });
    this.name = 'ApiContractError';
  }
}
