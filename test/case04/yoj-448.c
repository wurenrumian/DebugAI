#include <stdio.h>
#include <string.h>
int main()
{
    char x[100],y[100];
    int m,n,i;
    gets(x);
    scanf("\n");
    gets(y);
    m=strlen(x);
    n=strlen(y);
    for(i=0;i<=m;i++)
    {
        printf("%c",x[i]);
    }
    printf("\n");
    for(i=0;i<=n;i++)
    {
        printf("%c",y[i]);
    }
    printf("\n");
    return 0;
}