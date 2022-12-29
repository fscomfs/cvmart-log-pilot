#include <cuda_runtime.h>
#include <stdio.h>
#include <iostream>
#include <unistd.h>
#include <nvml.h>

int parseParameters(int argc, char *argv[],int &appNum,int &gpuInfo){
  int i = 0;
  if(argc == 1){
    return -1;
  }
  for (i = 1; i < argc; i++){
    if (strcmp(argv[i], "-appNum") == 0){
        appNum = (int)atoi(argv[++i]);
    }
    if (strcmp(argv[i], "-gpuInfo") == 0){
        gpuInfo = (int)atoi(argv[++i]);
    }
  }
  return 0;
}

__global__ void add(float* x, float * y, float* z, int n)
{
    // 获取全局索引
    int index = threadIdx.x + blockIdx.x * blockDim.x;
    // 步长
    int stride = blockDim.x * gridDim.x;
    for (int i = index; i < n; i += stride)
    {
        z[i] = x[i] + y[i];
    }
}

int checkGpuAvailability(int appNum){
    int deviceCount = 0;
    //0 成功 1内存溢出无法使用 2 进程冲突 3 检测异常 4 没有GPU 5 INIT_ERROR
    cudaError_t error_id = cudaGetDeviceCount(&deviceCount);
    if (error_id != cudaSuccess) {
      printf("result-cudaGetDeviceCount returned %d\n-> %s\n",
            static_cast<int>(error_id), cudaGetErrorString(error_id));
      std::cout <<"result-Init error "<<"初始化失败"<<std::endl;
      exit(5);
    }
    if (deviceCount==0)
    {  
       std::cout <<"result-cudaGetDeviceCount error "<<"没有检测到GPU"<<std::endl;
       exit(4);
    }
    
    int dev=0;
    int freeDeviceNum = 0;
    for (dev = 0; dev < deviceCount; ++dev) {
      cudaSetDevice(dev);
      size_t avail;
	    size_t total;
      cudaMemGetInfo(&avail, &total); 
      std::cout << "result-GPU-memory index-" <<dev<<":"<<(total-avail)/1024/1024<<"/"<<total/1024/1024<< std::endl;
      if (total==0)
      {
         std::cout <<"result-cudaMemGetInfo error index-"<<dev<<",无法获取设备内存信息"<<std::endl;
         exit(1);
      }
      if (total>0&&(total-avail)/total>0.1)
      { 
        //如果过使用量大于 0.1 个百分比 就认为当前这个显卡是有人在使用
      }else{
        freeDeviceNum++;
      }
    }
    if (freeDeviceNum<appNum)
    {
      std::cout << "result-CheckGpu error 存在进程冲突可能: " <<appNum<<"/"<< freeDeviceNum << std::endl;
      exit(2);
    }
    
    dev=0;
    for (dev = 0; dev < deviceCount; ++dev){
      std::cout << "cudaSetDevice: " <<dev<< std::endl;
      cudaError_t error_d = cudaSetDevice(dev);
      if (error_d != cudaSuccess) {
          std::cout <<"result-cudaSetDevice error index-"<<dev<<",errorInfo:"<<cudaGetErrorString(error_d)<<std::endl;
          exit(EXIT_FAILURE);
      }
      size_t avail;
	    size_t total;
      cudaMemGetInfo(&avail, &total); 
      if (total==0)
      {
         std::cout <<"result-cudaMemGetInfo error index-"<<dev<<",无法获取设备内存信息"<<std::endl;
         exit(EXIT_FAILURE);
      }
      if ((total-avail)/total>0.1)
      { 
          continue;
      }
      long N = 1 << 20;
      long nBytes = N * sizeof(float);
      // 申请host内存
      float *x, *y, *z;
      x = (float*)malloc(nBytes);
      y = (float*)malloc(nBytes);
      z = (float*)malloc(nBytes);

      // 初始化数据
      for (long i = 0; i < N; ++i)
      {
          x[i] = 10.0;
          y[i] = 20.0;
      }
      // 申请device内存
      float *d_x, *d_y, *d_z;
      //申请1G 的内存
      cudaError_t error_x  = cudaMalloc((void**)&d_x, nBytes);
      std::cout << "申请内存: " <<nBytes<< std::endl;
      if (error_x != cudaSuccess) {
          std::cout <<"result-cudaMalloc error index-"<<dev<<",errorInfo:"<<cudaGetErrorString(error_x)<<std::endl;
          exit(EXIT_FAILURE);
      }
      //申请1G 的内存
      cudaError_t error_y  = cudaMalloc((void**)&d_y, nBytes);
      if (error_y != cudaSuccess) {
          std::cout <<"result-cudaMalloc error index-"<<dev<<",errorInfo:"<<cudaGetErrorString(error_y)<<std::endl;
          exit(EXIT_FAILURE);
      }
      //申请1G 的内存
      cudaError_t error_z  = cudaMalloc((void**)&d_z, nBytes);
      if (error_z != cudaSuccess) {
          std::cout <<"result-cudaMalloc error index-"<<dev<<",errorInfo:"<<cudaGetErrorString(error_z)<<std::endl;
          exit(EXIT_FAILURE);
      }
      // 释放device内存
      cudaFree(d_x);
      cudaFree(d_y);
      cudaFree(d_z);
      // 释放host内存
      free(x);
      free(y);
      free(z);
    }
    return 0;
}


// int gpuInfo(int index){
//     // size_t avail;
// 	  // size_t total;
//     // cudaSetDevice(index);
//     // cudaMemGetInfo(&avail, &total); 
//     // std::cout <<"result-GPU-memory:"<<(total-avail)/1024/1024<<"/"<<total/1024/1024<<std::endl;
//     // sleep(5);
//     // return 0;

//    // nvmlInitWithFlags();
// }

int main(int argc, char *argv[])
{
    int appNum=0;
    int gpuInfoFlag = -1;
    if (-1==parseParameters(argc,argv,appNum,gpuInfoFlag))
    {
       printf("parameter_list failed!\n");
       exit(2);
    }
    printf("appNum:%d\n",appNum);
    printf("gpuInfoFlag:%d\n",gpuInfoFlag);
    return checkGpuAvailability(appNum);
}

