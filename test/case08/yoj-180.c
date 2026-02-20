#include<stdio.h>
#include<math.h>
int main() {
    float x,e,s=0,c=0;
    int i=1,j=0,k=1,p;
    long long m=1;
    scanf("%f %f",&x,&e);
    for (i=1; ;i=i+2){
        m=1;
     for(k=1;k<=i;k+=1){
        m=m*k;
     }
     if (i%4==3)
     p=(-1);
     else if (i%4==1)
     p=1;
     s=s+pow(x,i)*p/(m);

      if (fabs(sin(x)-s)<e)
      break;

    }
     printf("%f\n",s);
    for (j=0; ;j=j+2){
        m=1;
     if(j==0){
        m=1;
     }
     else if(j!=0){
        for(k=1;k<=j;k+=1){
        m=m*k;
     }
    }
     if (j%4==2)
     p=(-1);
     else if (j%4==0)
     p=1;
     c=c+pow(x,j)*p/(m);
      if (fabs(cos(x)-c)<e)
      break;
    }
     printf("%f",c);
     return 0;
    }