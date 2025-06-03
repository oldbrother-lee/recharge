export interface UserGrade {
  id: number;
  name: string;
  description: string;
  icon: string;
  grade_type: number;
  status: number;
  created_at: string;
  updated_at: string;
}

export interface UserGradeListRequest {
  page: number;
  page_size: number;
  name?: string;
  grade_type?: number;
  status?: number;
}

export type UserGradeListResponse = UserGrade[];

export interface UserGradeCreateRequest {
  name: string;
  description: string;
  icon: string;
  grade_type: number;
  status: number;
}

export interface UserGradeUpdateRequest extends UserGradeCreateRequest {
  id: number;
} 