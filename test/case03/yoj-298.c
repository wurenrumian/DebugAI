#include<stdio.h>
int main(){
    int a,b,x,y,z,c,d,e,f,g,h,j;
    scanf("%d %d %d %d %d",&a,&b,&x,&y,&z);
    for(int i=a;i<=b;i++){
        if(i%x==0&&i%y==0){
            c=i/1000000;
            i=i-c*1000000;
            d=i/100000;
            i-=d*100000;
            e=i/10000;
            i-=e*10000;
            f=i/1000;
            i-=f*1000;
            g=i/100;
            i-=g*100;
            h=i/10;
            i-=h*10;
            j=i;
            if(c==z||d==z||e==z||f==z||g==z||h==z||j==z) printf("%d\n",i);
            
        }
        
    }
    return 0;
}