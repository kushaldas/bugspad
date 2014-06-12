result = 1    
for i=1 to min{K,N-K}:
   result *= N-i+1
   result /= i
return result
