#include <stdio.h>
#include <math.h>
int main(){
    int a;
    float b;
    scanf("%d",&a);
    if(a>=90)
{
		b=4.0;}
	else if(86<=a) 
{
		b=3.7;}
	else if(83<=a)
{
		b=3.3;}
	else if(80<=a)
{
		b=3.0;}
	else if(76<=a)
{
		b=2.7;}
	else if(73<=a)
{
		b=2.3;}
	else if(70<=a<=72)
{
		b=2.0;}
	else if(66<=a<=69)
{
		b=1.7;}
	else if(63<=a<=65)
{
		b=1.3;}
	else if(60<=a<=62)
{
		b=1.0;}
	else 
{
		b=0;}
	printf("%.1f",b);
    return 0;
}