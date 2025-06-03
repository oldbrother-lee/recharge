import { request } from '@/service/request';
import type { UserGrade, UserGradeListRequest, UserGradeListResponse, UserGradeCreateRequest, UserGradeUpdateRequest } from '@/typings/api/user-grade';

export const getUserGradeList = (params: UserGradeListRequest) => {
  return request({
    url: '/user-grades/list',
    method: 'GET',
    params: {
      current: params.page,
      size: params.page_size,
      name: params.name,
      grade_type: params.grade_type,
      status: params.status
    }
  });
};

export const createUserGrade = (data: UserGradeCreateRequest) => {
  return request({
    url: '/user-grades',
    method: 'POST',
    data
  });
};

export const updateUserGrade = (data: UserGradeUpdateRequest) => {
  return request({
    url: `/user-grades/${data.id}`,
    method: 'PUT',
    data
  });
};

export const deleteUserGrade = (id: number) => {
  return request({
    url: `/user-grades/${id}`,
    method: 'DELETE'
  });
}; 