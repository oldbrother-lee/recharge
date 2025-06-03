package model

// ProductListRequest 商品列表请求
type ProductListRequest struct {
	Page     int    `form:"page" binding:"required,min=1"`
	PageSize int    `form:"page_size" binding:"required,min=1,max=100"`
	Type     int    `form:"type"`
	Category int    `form:"category"`
	ISP      string `form:"isp"`
	Status   int    `form:"status"`
}

// ProductListResponse 商品列表响应
type ProductListResponse struct {
	Total   int64      `json:"total"`
	Records []*Product `json:"records"`
}

// ProductCategoryListResponse 商品分类列表响应
type ProductCategoryListResponse struct {
	Total int64             `json:"total"`
	List  []ProductCategory `json:"list"`
}

// ProductDetailResponse 商品详情响应
type ProductDetailResponse struct {
	Product     Product             `json:"product"`
	Specs       []ProductSpec       `json:"specs"`
	GradePrices []ProductGradePrice `json:"grade_prices"`
	Category    ProductCategory     `json:"category"`
}

// ProductCreateRequest 创建商品请求
type ProductCreateRequest struct {
	Name            string  `json:"name" binding:"required"`
	Description     string  `json:"description"`
	Price           float64 `json:"price" binding:"required"`
	Type            int     `json:"type" binding:"required"`
	ISP             string  `json:"isp"`
	Status          int     `json:"status"`
	Sort            int     `json:"sort"`
	APIEnabled      bool    `json:"api_enabled"`
	Remark          string  `json:"remark"`
	CategoryID      int64   `json:"category_id" binding:"required"`
	OperatorTag     string  `json:"operator_tag"`
	MaxPrice        float64 `json:"max_price"`
	VoucherPrice    string  `json:"voucher_price"`
	VoucherName     string  `json:"voucher_name"`
	ShowStyle       int     `json:"show_style"`
	APIFailStyle    int     `json:"api_fail_style"`
	AllowProvinces  string  `json:"allow_provinces"`
	AllowCities     string  `json:"allow_cities"`
	ForbidProvinces string  `json:"forbid_provinces"`
	ForbidCities    string  `json:"forbid_cities"`
	APIDelay        string  `json:"api_delay"`
	GradeIDs        string  `json:"grade_ids"`
	APIID           int64   `json:"api_id"`
	APIParamID      int64   `json:"api_param_id"`
	IsApi           bool    `json:"is_api"`
}

// ProductUpdateRequest 更新商品请求
type ProductUpdateRequest struct {
	ID              int64   `json:"id" binding:"required"`
	Name            string  `json:"name" binding:"required"`
	Description     string  `json:"description"`
	Price           float64 `json:"price" binding:"required"`
	Type            int     `json:"type" binding:"required"`
	ISP             string  `json:"isp"`
	Status          int     `json:"status"`
	Sort            int     `json:"sort"`
	APIEnabled      bool    `json:"api_enabled"`
	Remark          string  `json:"remark"`
	CategoryID      int64   `json:"category_id" binding:"required"`
	OperatorTag     string  `json:"operator_tag"`
	MaxPrice        float64 `json:"max_price"`
	VoucherPrice    string  `json:"voucher_price"`
	VoucherName     string  `json:"voucher_name"`
	ShowStyle       int     `json:"show_style"`
	APIFailStyle    int     `json:"api_fail_style"`
	AllowProvinces  string  `json:"allow_provinces"`
	AllowCities     string  `json:"allow_cities"`
	ForbidProvinces string  `json:"forbid_provinces"`
	ForbidCities    string  `json:"forbid_cities"`
	APIDelay        string  `json:"api_delay"`
	GradeIDs        string  `json:"grade_ids"`
	APIID           int64   `json:"api_id"`
	APIParamID      int64   `json:"api_param_id"`
	IsApi           bool    `json:"is_api"`
}

// ProductAPIRelationCreateRequest 创建商品接口关联请求
type ProductAPIRelationCreateRequest struct {
	ProductID int64  `json:"product_id" binding:"required"`
	APIID     int64  `json:"api_id" binding:"required"`
	ParamID   int64  `json:"param_id" binding:"required"`
	Sort      int    `json:"sort"`
	Status    int    `json:"status" binding:"oneof=0 1"`
	RetryNum  int    `json:"retry_num"`
	Isp       string `json:"isp" binding:"required"`
}

// ProductAPIRelationUpdateRequest 更新商品接口关联请求
type ProductAPIRelationUpdateRequest struct {
	ID        int64  `json:"id" binding:"required"`
	ProductID int64  `json:"product_id" binding:"required"`
	APIID     int64  `json:"api_id" binding:"required"`
	ParamID   int64  `json:"param_id" binding:"required"`
	Sort      int    `json:"sort"`
	Status    int    `json:"status" binding:"oneof=0 1"`
	RetryNum  int    `json:"retry_num"`
	Isp       string `json:"isp" binding:"required"`
}

// ProductAPIRelationListRequest 获取商品接口关联列表请求
type ProductAPIRelationListRequest struct {
	Page      int    `form:"page" binding:"required,min=1"`
	PageSize  int    `form:"page_size" binding:"required,min=1,max=100"`
	ProductID *int64 `form:"product_id"`
	APIID     *int64 `form:"api_id"`
	Status    *int   `form:"status"`
}

// ProductAPIRelationListResponse 获取商品接口关联列表响应
type ProductAPIRelationListResponse struct {
	Total int64                `json:"total"`
	List  []ProductAPIRelation `json:"list"`
}
